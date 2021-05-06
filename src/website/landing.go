package website

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

type LandingTemplateData struct {
	templates.BaseData

	NewsPost             LandingPageFeaturedPost
	PostColumns          [][]LandingPageProject
	ShowcaseTimelineJson string
}

type LandingPageProject struct {
	Project      templates.Project
	FeaturedPost *LandingPageFeaturedPost
	Posts        []templates.PostListItem
}

type LandingPageFeaturedPost struct {
	Title   string
	Url     string
	User    templates.User
	Date    time.Time
	Unread  bool
	Content template.HTML
}

func Index(c *RequestContext) ResponseData {
	const maxPosts = 5
	const numProjectsToGet = 7

	c.Perf.StartBlock("SQL", "Fetch projects")
	iterProjects, err := db.Query(c.Context(), c.Conn, models.Project{},
		`
		SELECT $columns
		FROM handmade_project
		WHERE
			flags = 0
			OR id = $1
		ORDER BY all_last_updated DESC
		LIMIT $2
		`,
		models.HMNProjectID,
		numProjectsToGet*2, // hedge your bets against projects that don't have any content
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get projects for home page"))
	}
	defer iterProjects.Close()

	var pageProjects []LandingPageProject

	allProjects := iterProjects.ToSlice()
	c.Perf.EndBlock()
	c.Logger.Debug().Interface("allProjects", allProjects).Msg("all the projects")

	categoryUrls := GetAllCategoryUrls(c.Context(), c.Conn)

	var currentUserId *int
	if c.CurrentUser != nil {
		currentUserId = &c.CurrentUser.ID
	}

	c.Perf.StartBlock("LANDING", "Process projects")
	for _, projRow := range allProjects {
		proj := projRow.(*models.Project)

		c.Perf.StartBlock("SQL", fmt.Sprintf("Fetch posts for %s", *proj.Name))
		type projectPostQuery struct {
			Post               models.Post   `db:"post"`
			Thread             models.Thread `db:"thread"`
			User               models.User   `db:"auth_user"`
			ThreadLastReadTime *time.Time    `db:"tlri.lastread"`
			CatLastReadTime    *time.Time    `db:"clri.lastread"`
		}
		projectPostIter, err := db.Query(c.Context(), c.Conn, projectPostQuery{},
			`
			SELECT $columns
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
				LEFT JOIN handmade_threadlastreadinfo AS tlri ON (
					tlri.thread_id = post.thread_id
					AND tlri.user_id = $1
				)
				LEFT JOIN handmade_categorylastreadinfo AS clri ON (
					clri.category_id = post.category_id
					AND clri.user_id = $1
				)
				LEFT JOIN auth_user ON post.author_id = auth_user.id
			WHERE
				post.project_id = $2
				AND post.category_kind IN ($3, $4, $5, $6)
				AND post.deleted = FALSE
			ORDER BY postdate DESC
			LIMIT $7
			`,
			currentUserId,
			proj.ID,
			models.CatKindBlog, models.CatKindForum, models.CatKindWiki, models.CatKindLibraryResource,
			maxPosts,
		)
		c.Perf.EndBlock()
		if err != nil {
			c.Logger.Error().Err(err).Msg("failed to fetch project posts")
			continue
		}
		projectPosts := projectPostIter.ToSlice()

		landingPageProject := LandingPageProject{
			Project: templates.ProjectToTemplate(proj),
		}

		for _, projectPostRow := range projectPosts {
			projectPost := projectPostRow.(*projectPostQuery)

			hasRead := false
			if projectPost.ThreadLastReadTime != nil && projectPost.ThreadLastReadTime.After(projectPost.Post.PostDate) {
				hasRead = true
			} else if projectPost.CatLastReadTime != nil && projectPost.CatLastReadTime.After(projectPost.Post.PostDate) {
				hasRead = true
			}

			featurable := (!proj.IsHMN() &&
				projectPost.Post.CategoryKind == models.CatKindBlog &&
				projectPost.Post.ParentID == nil &&
				landingPageProject.FeaturedPost == nil)

			if featurable {
				type featuredContentResult struct {
					Content string `db:"ver.text_parsed"`
				}

				c.Perf.StartBlock("SQL", "Fetch featured post content")
				contentResult, err := db.QueryOne(c.Context(), c.Conn, featuredContentResult{}, `
					SELECT $columns
					FROM
						handmade_post AS post
						JOIN handmade_postversion AS ver ON post.current_id = ver.id
					WHERE
						post.id = $1
				`, projectPost.Post.ID)
				if err != nil {
					panic(err)
				}
				c.Perf.EndBlock()
				content := contentResult.(*featuredContentResult).Content

				landingPageProject.FeaturedPost = &LandingPageFeaturedPost{
					Title:   projectPost.Thread.Title,
					Url:     PostUrl(projectPost.Post, projectPost.Post.CategoryKind, categoryUrls[projectPost.Post.CategoryID]),
					User:    templates.UserToTemplate(&projectPost.User),
					Date:    projectPost.Post.PostDate,
					Unread:  !hasRead,
					Content: template.HTML(content),
				}
			} else {
				landingPageProject.Posts = append(landingPageProject.Posts, templates.PostListItem{
					Title:  projectPost.Thread.Title,
					Url:    PostUrl(projectPost.Post, projectPost.Post.CategoryKind, categoryUrls[projectPost.Post.CategoryID]),
					User:   templates.UserToTemplate(&projectPost.User),
					Date:   projectPost.Post.PostDate,
					Unread: !hasRead,
				})
			}
		}

		if len(projectPosts) > 0 {
			pageProjects = append(pageProjects, landingPageProject)
		}

		if len(pageProjects) >= numProjectsToGet {
			break
		}
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Get news")
	type newsThreadQuery struct {
		Thread models.Thread `db:"thread"`
	}
	newsThreadRow, err := db.QueryOne(c.Context(), c.Conn, newsThreadQuery{},
		`
		SELECT $columns
		FROM
			handmade_thread as thread
			JOIN handmade_category AS cat ON thread.category_id = cat.id
		WHERE
			cat.project_id = $1
			AND cat.kind = $2
		`,
		models.HMNProjectID,
		models.CatKindBlog,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch latest news post"))
	}
	c.Perf.EndBlock()
	newsThread := newsThreadRow.(*newsThreadQuery)
	_ = newsThread // TODO: NO

	/*
		Columns are filled by placing projects into the least full column.
		The fill array tracks the estimated sizes.

		This is all hardcoded for two columns; deal with it.
	*/
	cols := [][]LandingPageProject{nil, nil}
	fill := []int{4, 0}
	featuredIndex := []int{0, 0}
	for _, pageProject := range pageProjects {
		leastFullColumnIndex := indexOfSmallestInt(fill)

		numNewPosts := len(pageProject.Posts)
		if numNewPosts > maxPosts {
			numNewPosts = maxPosts
		}

		fill[leastFullColumnIndex] += numNewPosts

		if pageProject.FeaturedPost != nil {
			fill[leastFullColumnIndex] += 2 // featured posts add more to height

			// projects with featured posts go at the top of the column
			cols[leastFullColumnIndex] = append(cols[leastFullColumnIndex], pageProject)
			featuredIndex[leastFullColumnIndex] += 1
		} else {
			cols[leastFullColumnIndex] = append(cols[leastFullColumnIndex], pageProject)
		}
	}

	type newsPostQuery struct {
		Post        models.Post        `db:"post"`
		PostVersion models.PostVersion `db:"ver"`
		Thread      models.Thread      `db:"thread"`
		User        models.User        `db:"auth_user"`
	}
	newsPostRow, err := db.QueryOne(c.Context(), c.Conn, newsPostQuery{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			JOIN handmade_thread AS thread ON post.thread_id = thread.id
			JOIN handmade_category AS cat ON thread.category_id = cat.id
			JOIN auth_user ON post.author_id = auth_user.id
			JOIN handmade_postversion AS ver ON post.current_id = ver.id
		WHERE
			cat.project_id = $1
			AND cat.kind = $2
			AND post.id = thread.first_id
			AND NOT thread.deleted
		ORDER BY post.postdate DESC
		LIMIT 1
		`,
		models.HMNProjectID,
		models.CatKindBlog,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch news post"))
	}
	newsPostResult := newsPostRow.(*newsPostQuery)

	baseData := getBaseData(c)
	baseData.BodyClasses = append(baseData.BodyClasses, "hmdev", "landing") // TODO: Is "hmdev" necessary any more?

	var res ResponseData
	err = res.WriteTemplate("landing.html", LandingTemplateData{
		BaseData: baseData,
		NewsPost: LandingPageFeaturedPost{
			Title:   newsPostResult.Thread.Title,
			Url:     PostUrl(newsPostResult.Post, models.CatKindBlog, ""),
			User:    templates.UserToTemplate(&newsPostResult.User),
			Date:    newsPostResult.Post.PostDate,
			Unread:  true, // TODO
			Content: template.HTML(newsPostResult.PostVersion.TextParsed),
		},
		PostColumns: cols,
	}, c.Perf)
	if err != nil {
		panic(err)
	}

	return res
}

func indexOfSmallestInt(s []int) int {
	result := 0
	min := s[result]

	for i, val := range s {
		if val < min {
			result = i
			min = val
		}
	}

	return result
}

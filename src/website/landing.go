package website

import (
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

type LandingTemplateData struct {
	templates.BaseData

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
	Content string
}

func Index(c *RequestContext) ResponseData {
	const maxPosts = 5
	const numProjectsToGet = 7

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
	c.Logger.Info().Interface("allProjects", allProjects).Msg("all the projects")

	var currentUserId *int
	if c.CurrentUser != nil {
		currentUserId = &c.CurrentUser.ID
	}

	for _, projRow := range allProjects {
		proj := projRow.(*models.Project)

		type ProjectPost struct {
			Post               models.Post     `db:"post"`
			Thread             models.Thread   `db:"thread"`
			Cat                models.Category `db:"cat"`
			User               models.User     `db:"auth_user"`
			ThreadLastReadTime *time.Time      `db:"tlri.lastread"`
			CatLastReadTime    *time.Time      `db:"clri.lastread"`
		}

		projectPostIter, err := db.Query(c.Context(), c.Conn, ProjectPost{},
			`
			SELECT $columns
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON thread.id = post.thread_id
				JOIN handmade_category AS cat ON cat.id = thread.category_id
				LEFT OUTER JOIN handmade_threadlastreadinfo AS tlri ON (
					tlri.thread_id = thread.id
					AND tlri.user_id = $1
				)
				LEFT OUTER JOIN handmade_categorylastreadinfo AS clri ON (
					clri.category_id = cat.id
					AND clri.user_id = $1
				)
				LEFT OUTER JOIN auth_user ON post.author_id = auth_user.id
			WHERE
				cat.project_id = $2
				AND cat.kind IN ($3, $4, $5, $6)
				AND post.moderated = FALSE
				AND post.thread_id IS NOT NULL
			ORDER BY postdate DESC
			LIMIT $7
			`,
			currentUserId,
			proj.ID,
			models.CatTypeBlog, models.CatTypeForum, models.CatTypeWiki, models.CatTypeLibraryResource,
			maxPosts,
		)
		if err != nil {
			c.Logger.Error().Err(err).Msg("failed to fetch project posts")
			continue
		}
		projectPosts := projectPostIter.ToSlice()

		landingPageProject := LandingPageProject{
			Project: templates.ProjectToTemplate(proj),
		}

		for _, projectPostRow := range projectPosts {
			projectPost := projectPostRow.(*ProjectPost)

			hasRead := false
			if projectPost.ThreadLastReadTime != nil && projectPost.ThreadLastReadTime.After(projectPost.Post.PostDate) {
				hasRead = true
			} else if projectPost.CatLastReadTime != nil && projectPost.CatLastReadTime.After(projectPost.Post.PostDate) {
				hasRead = true
			}

			featurable := (!proj.IsHMN() &&
				projectPost.Cat.Kind == models.CatTypeBlog &&
				projectPost.Post.ParentID == nil &&
				landingPageProject.FeaturedPost == nil)

			if featurable {
				type featuredContentResult struct {
					Content string `db:"ver.text_parsed"`
				}

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
				content := contentResult.(*featuredContentResult).Content

				// c.Logger.Debug().Str("content", content).Msg("")

				landingPageProject.FeaturedPost = &LandingPageFeaturedPost{
					Title:   projectPost.Thread.Title,
					Url:     templates.PostUrl(projectPost.Post, projectPost.Cat.Kind, proj.Subdomain()), // TODO
					User:    templates.UserToTemplate(&projectPost.User),
					Date:    projectPost.Post.PostDate,
					Unread:  !hasRead,
					Content: content,
				}
			} else {
				landingPageProject.Posts = append(landingPageProject.Posts, templates.PostListItem{
					Title:  projectPost.Thread.Title,
					Url:    templates.PostUrl(projectPost.Post, projectPost.Cat.Kind, proj.Subdomain()), // TODO
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
		models.CatTypeBlog,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch latest news post"))
	}
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

	baseData := getBaseData(c)
	baseData.BodyClasses = append(baseData.BodyClasses, "hmdev", "landing") // TODO: Is "hmdev" necessary any more?

	var res ResponseData
	err = res.WriteTemplate("index.html", LandingTemplateData{
		BaseData:    getBaseData(c),
		PostColumns: cols,
	})
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

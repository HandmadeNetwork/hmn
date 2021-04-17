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
	FeaturedPost *LandingPagePost
	Posts        []LandingPagePost
}

type LandingPagePost struct {
	Post    templates.Post
	HasRead bool
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
			Post               models.Post `db:"post"`
			ThreadLastReadTime *time.Time  `db:"tlri.lastread"`
			CatLastReadTime    *time.Time  `db:"clri.lastread"`
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

			landingPageProject.Posts = append(landingPageProject.Posts, LandingPagePost{
				Post:    templates.PostToTemplate(&projectPost.Post),
				HasRead: hasRead,
			})
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

	baseData := getBaseData(c)
	baseData.BodyClasses = append(baseData.BodyClasses, "hmdev", "landing") // TODO: Is "hmdev" necessary any more?

	var res ResponseData
	err = res.WriteTemplate("index.html", LandingTemplateData{
		BaseData:    getBaseData(c),
		PostColumns: [][]LandingPageProject{pageProjects}, // TODO: NO
	})
	if err != nil {
		panic(err)
	}

	return res
}

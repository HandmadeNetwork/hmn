package website

import (
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/julienschmidt/httprouter"
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

func (s *websiteRoutes) Index(c *RequestContext, p httprouter.Params) {
	const maxPosts = 5
	const numProjectsToGet = 7

	iterProjects, err := db.Query(c.Context(), s.conn, models.Project{},
		"SELECT $columns FROM handmade_project WHERE flags = 0 OR id = $1",
		models.HMNProjectID,
	)
	if err != nil {
		c.Errored(http.StatusInternalServerError, oops.New(err, "failed to get projects for home page"))
		return
	}
	defer iterProjects.Close()

	var pageProjects []LandingPageProject
	_ = pageProjects // TODO: NO

	for _, projRow := range iterProjects.ToSlice() {
		proj := projRow.(*models.Project)

		type ProjectPost struct {
			Post               models.Post `db:"post"`
			ThreadLastReadTime *time.Time  `db:"tlri.lastread"`
			CatLastReadTime    *time.Time  `db:"clri.lastread"`
		}

		memberId := 3 // TODO: NO
		projectPostIter, err := db.Query(c.Context(), s.conn, ProjectPost{},
			`
			SELECT $columns
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON thread.id = post.thread_id
				JOIN handmade_category AS cat ON cat.id = thread.category_id
				LEFT OUTER JOIN handmade_threadlastreadinfo AS tlri ON (
					tlri.thread_id = thread.id
					AND tlri.member_id = $1
				)
				LEFT OUTER JOIN handmade_categorylastreadinfo AS clri ON (
					clri.category_id = cat.id
					AND clri.member_id = $1
				)
			WHERE
				cat.project_id = $2
				AND cat.kind IN ($3, $4, $5, $6)
				AND post.moderated = FALSE
				AND post.thread_id IS NOT NULL
			ORDER BY postdate DESC
			LIMIT $7
			`,
			memberId,
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
			Project: templates.Project{ // TODO: Use a common function to map from model to template data
				Name:      *proj.Name,
				Subdomain: *proj.Slug,
				// ...
			},
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
				Post:    templates.Post{}, // TODO: Use a common function to map from model to template again
				HasRead: hasRead,
			})
		}
	}

	type newsThreadQuery struct {
		Thread models.Thread `db:"thread"`
	}
	newsThreadRow, err := db.QueryOne(c.Context(), s.conn, newsThreadQuery{},
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
		c.Errored(http.StatusInternalServerError, oops.New(err, "failed to fetch latest news post"))
		return
	}
	newsThread := newsThreadRow.(*newsThreadQuery)
	_ = newsThread // TODO: NO

	baseData := s.getBaseData(c)
	baseData.BodyClasses = append(baseData.BodyClasses, "hmdev", "landing") // TODO: Is "hmdev" necessary any more?

	err = c.WriteTemplate("index.html", s.getBaseData(c))
	if err != nil {
		panic(err)
	}
}

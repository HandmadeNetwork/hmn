package website

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

type ProjectTemplateData struct {
	templates.BaseData

	Pagination       templates.Pagination
	CarouselProjects []templates.Project
	Projects         []templates.Project
	PersonalProjects []templates.Project

	ProjectAtomFeedUrl string
	WIPForumUrl        string
}

func ProjectIndex(c *RequestContext) ResponseData {
	const projectsPerPage = 20
	const maxCarouselProjects = 10
	const maxPersonalProjects = 10

	officialProjects, err := FetchProjects(c.Context(), c.Conn, c.CurrentUser, ProjectsQuery{
		Types: OfficialProjects,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch projects"))
	}

	numPages := int(math.Ceil(float64(len(officialProjects)) / projectsPerPage))
	page, numPages, ok := getPageInfo(c.PathParams["page"], len(officialProjects), feedPostsPerPage)
	if !ok {
		return c.Redirect(hmnurl.BuildProjectIndex(1), http.StatusSeeOther)
	}

	pagination := templates.Pagination{
		Current: page,
		Total:   numPages,

		FirstUrl:    hmnurl.BuildProjectIndex(1),
		LastUrl:     hmnurl.BuildProjectIndex(numPages),
		NextUrl:     hmnurl.BuildProjectIndex(utils.IntClamp(1, page+1, numPages)),
		PreviousUrl: hmnurl.BuildProjectIndex(utils.IntClamp(1, page-1, numPages)),
	}

	c.Perf.StartBlock("PROJECTS", "Grouping and sorting")
	var handmadeHero *templates.Project
	var featuredProjects []templates.Project
	var recentProjects []templates.Project
	var restProjects []templates.Project
	now := time.Now()
	for _, p := range officialProjects {
		templateProject := templates.ProjectToTemplate(&p.Project, c.Theme)
		if p.Project.Slug == "hero" {
			// NOTE(asaf): Handmade Hero gets special treatment. Must always be first in the list.
			handmadeHero = &templateProject
			continue
		}
		if p.Project.Featured {
			featuredProjects = append(featuredProjects, templateProject)
		} else if now.Sub(p.Project.AllLastUpdated).Seconds() < models.RecentProjectUpdateTimespanSec {
			recentProjects = append(recentProjects, templateProject)
		} else {
			restProjects = append(restProjects, templateProject)
		}
	}

	_, randSeed := now.ISOWeek()
	random := rand.New(rand.NewSource(int64(randSeed)))
	random.Shuffle(len(featuredProjects), func(i, j int) { featuredProjects[i], featuredProjects[j] = featuredProjects[j], featuredProjects[i] })
	random.Shuffle(len(recentProjects), func(i, j int) { recentProjects[i], recentProjects[j] = recentProjects[j], recentProjects[i] })
	random.Shuffle(len(restProjects), func(i, j int) { restProjects[i], restProjects[j] = restProjects[j], restProjects[i] })

	if handmadeHero != nil {
		// NOTE(asaf): As mentioned above, inserting HMH first.
		featuredProjects = append([]templates.Project{*handmadeHero}, featuredProjects...)
	}

	orderedProjects := make([]templates.Project, 0, len(featuredProjects)+len(recentProjects)+len(restProjects))
	orderedProjects = append(orderedProjects, featuredProjects...)
	orderedProjects = append(orderedProjects, recentProjects...)
	orderedProjects = append(orderedProjects, restProjects...)

	firstProjectIndex := (page - 1) * projectsPerPage
	endIndex := utils.IntMin(firstProjectIndex+projectsPerPage, len(orderedProjects))
	pageProjects := orderedProjects[firstProjectIndex:endIndex]

	var carouselProjects []templates.Project
	if page == 1 {
		carouselProjects = featuredProjects[:utils.IntMin(len(featuredProjects), maxCarouselProjects)]
	}
	c.Perf.EndBlock()

	// Fetch and highlight a random selection of personal projects
	var personalProjects []templates.Project
	{
		projects, err := FetchProjects(c.Context(), c.Conn, c.CurrentUser, ProjectsQuery{
			Types: PersonalProjects,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch personal projects"))
		}

		randSeed := now.YearDay()
		random := rand.New(rand.NewSource(int64(randSeed)))
		random.Shuffle(len(projects), func(i, j int) { projects[i], projects[j] = projects[j], projects[i] })

		for i, p := range projects {
			if i >= maxPersonalProjects {
				break
			}
			personalProjects = append(personalProjects, templates.ProjectToTemplate(&p.Project, c.Theme))
		}
	}

	baseData := getBaseDataAutocrumb(c, "Projects")
	var res ResponseData
	res.MustWriteTemplate("project_index.html", ProjectTemplateData{
		BaseData: baseData,

		Pagination:       pagination,
		CarouselProjects: carouselProjects,
		Projects:         pageProjects,
		PersonalProjects: personalProjects,

		ProjectAtomFeedUrl: hmnurl.BuildAtomFeedForProjects(),
		WIPForumUrl:        hmnurl.BuildForum(models.HMNProjectSlug, []string{"wip"}, 1),
	}, c.Perf)
	return res
}

type ProjectHomepageData struct {
	templates.BaseData
	Project        templates.Project
	Owners         []templates.User
	Screenshots    []string
	ProjectLinks   []templates.Link
	Licenses       []templates.Link
	RecentActivity []templates.TimelineItem
}

func ProjectHomepage(c *RequestContext) ResponseData {
	maxRecentActivity := 15
	var project *models.Project

	if c.CurrentProject.IsHMN() {
		slug, hasSlug := c.PathParams["slug"]
		if hasSlug && slug != "" {
			slug = strings.ToLower(slug)
			if slug == models.HMNProjectSlug {
				return c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
			}
			c.Perf.StartBlock("SQL", "Fetching project by slug")
			type projectQuery struct {
				Project models.Project `db:"Project"`
			}
			projectQueryResult, err := db.QueryOne(c.Context(), c.Conn, projectQuery{},
				`
					SELECT $columns
					FROM
						handmade_project AS project
					WHERE
						LOWER(project.slug) = $1
				`,
				slug,
			)
			c.Perf.EndBlock()
			if err != nil {
				if errors.Is(err, db.NotFound) {
					return FourOhFour(c)
				} else {
					return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project by slug"))
				}
			}
			project = &projectQueryResult.(*projectQuery).Project
			if project.Lifecycle != models.ProjectLifecycleUnapproved && project.Lifecycle != models.ProjectLifecycleApprovalRequired {
				return c.Redirect(hmnurl.BuildProjectHomepage(project.Slug), http.StatusSeeOther)
			}
		}
	} else {
		project = c.CurrentProject
	}

	if project == nil {
		return FourOhFour(c)
	}

	owners, err := FetchProjectOwners(c.Context(), c.Conn, project.ID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	canView := false
	canEdit := false
	if c.CurrentUser != nil {
		if c.CurrentUser.IsStaff {
			canView = true
			canEdit = true
		} else {
			for _, owner := range owners {
				if owner.ID == c.CurrentUser.ID {
					canView = true
					canEdit = true
					break
				}
			}
		}
	}
	if !canView {
		if !project.Hidden {
			for _, lc := range models.VisibleProjectLifecycles {
				if project.Lifecycle == lc {
					canView = true
					break
				}
			}
		}
	}

	if !canView {
		return FourOhFour(c)
	}

	c.Perf.StartBlock("SQL", "Fetching screenshots")
	type screenshotQuery struct {
		Filename string `db:"screenshot.file"`
	}
	screenshotQueryResult, err := db.Query(c.Context(), c.Conn, screenshotQuery{},
		`
		SELECT $columns
		FROM
			handmade_imagefile AS screenshot
			INNER JOIN handmade_project_screenshots ON screenshot.id = handmade_project_screenshots.imagefile_id
		WHERE
			handmade_project_screenshots.project_id = $1
		`,
		project.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch screenshots for project"))
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetching project links")
	type projectLinkQuery struct {
		Link models.Link `db:"link"`
	}
	projectLinkResult, err := db.Query(c.Context(), c.Conn, projectLinkQuery{},
		`
		SELECT $columns
		FROM
			handmade_links as link
		WHERE
			link.project_id = $1
		ORDER BY link.ordering ASC
		`,
		project.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project links"))
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetching project timeline")
	type postQuery struct {
		Post   models.Post   `db:"post"`
		Thread models.Thread `db:"thread"`
		Author models.User   `db:"author"`
	}
	postQueryResult, err := db.Query(c.Context(), c.Conn, postQuery{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			INNER JOIN handmade_thread AS thread ON thread.id = post.thread_id
			INNER JOIN auth_user AS author ON author.id = post.author_id
		WHERE
			post.project_id = $1
		ORDER BY post.postdate DESC
		LIMIT $2
		`,
		project.ID,
		maxRecentActivity,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project posts"))
	}
	c.Perf.EndBlock()

	var projectHomepageData ProjectHomepageData

	projectHomepageData.BaseData = getBaseData(c, project.Name, nil)
	if canEdit {
		// TODO: Move to project-specific navigation
		// projectHomepageData.BaseData.Header.EditURL = hmnurl.BuildProjectEdit(project.Slug, "")
	}
	projectHomepageData.BaseData.OpenGraphItems = append(projectHomepageData.BaseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    project.Blurb,
	})

	projectHomepageData.Project = templates.ProjectToTemplate(project, c.Theme)
	for _, owner := range owners {
		projectHomepageData.Owners = append(projectHomepageData.Owners, templates.UserToTemplate(owner, c.Theme))
	}

	if project.Hidden {
		projectHomepageData.BaseData.AddImmediateNotice(
			"hidden",
			"NOTICE: This project is hidden. It is currently visible only to owners and site admins.",
		)
	}

	if project.Lifecycle != models.ProjectLifecycleActive {
		switch project.Lifecycle {
		case models.ProjectLifecycleUnapproved:
			projectHomepageData.BaseData.AddImmediateNotice(
				"unapproved",
				fmt.Sprintf(
					"NOTICE: This project has not yet been submitted for approval. It is only visible to owners. Please <a href=\"%s\">submit it for approval</a> when the project content is ready for review.",
					hmnurl.BuildProjectEdit(project.Slug, "submit"),
				),
			)
		case models.ProjectLifecycleApprovalRequired:
			projectHomepageData.BaseData.AddImmediateNotice(
				"unapproved",
				"NOTICE: This project is awaiting approval. It is only visible to owners and site admins.",
			)
		case models.ProjectLifecycleHiatus:
			projectHomepageData.BaseData.AddImmediateNotice(
				"hiatus",
				"NOTICE: This project is on hiatus and may not update for a while.",
			)
		case models.ProjectLifecycleDead:
			projectHomepageData.BaseData.AddImmediateNotice(
				"dead",
				"NOTICE: Site staff have marked this project as being dead. If you intend to revive it, please contact a member of the Handmade Network staff.",
			)
		case models.ProjectLifecycleLTSRequired:
			projectHomepageData.BaseData.AddImmediateNotice(
				"lts-reqd",
				"NOTICE: This project is awaiting approval for maintenance-mode status.",
			)
		case models.ProjectLifecycleLTS:
			projectHomepageData.BaseData.AddImmediateNotice(
				"lts",
				"NOTICE: This project has reached a state of completion.",
			)
		}
	}

	for _, screenshot := range screenshotQueryResult.ToSlice() {
		projectHomepageData.Screenshots = append(projectHomepageData.Screenshots, hmnurl.BuildUserFile(screenshot.(*screenshotQuery).Filename))
	}

	for _, link := range projectLinkResult.ToSlice() {
		projectHomepageData.ProjectLinks = append(projectHomepageData.ProjectLinks, templates.LinkToTemplate(&link.(*projectLinkQuery).Link))
	}

	for _, post := range postQueryResult.ToSlice() {
		projectHomepageData.RecentActivity = append(projectHomepageData.RecentActivity, PostToTimelineItem(
			lineageBuilder,
			&post.(*postQuery).Post,
			&post.(*postQuery).Thread,
			project,
			&post.(*postQuery).Author,
			c.Theme,
		))
	}

	var res ResponseData
	err = res.WriteTemplate("project_homepage.html", projectHomepageData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render project homepage template"))
	}
	return res
}

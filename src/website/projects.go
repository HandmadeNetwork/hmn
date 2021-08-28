package website

import (
	"errors"
	"fmt"
	"math"
	"math/rand"
	"net/http"
	"strconv"
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

	UserPendingProjectUnderReview bool
	UserPendingProject            *templates.Project
	UserApprovedProjects          []templates.Project

	ProjectAtomFeedUrl string
	ManifestoUrl       string
	NewProjectUrl      string
	RegisterUrl        string
	LoginUrl           string
}

func ProjectIndex(c *RequestContext) ResponseData {
	const projectsPerPage = 20
	const maxCarouselProjects = 10

	page := 1
	pageString, hasPage := c.PathParams["page"]
	if hasPage && pageString != "" {
		if pageParsed, err := strconv.Atoi(pageString); err == nil {
			page = pageParsed
		} else {
			return c.Redirect(hmnurl.BuildProjectIndex(1), http.StatusSeeOther)
		}
	}

	if page < 1 {
		return c.Redirect(hmnurl.BuildProjectIndex(1), http.StatusSeeOther)
	}

	c.Perf.StartBlock("SQL", "Fetching all visible projects")
	type projectResult struct {
		Project models.Project `db:"project"`
	}
	allProjects, err := db.Query(c.Context(), c.Conn, projectResult{},
		`
		SELECT $columns
		FROM
			handmade_project AS project
		WHERE
			project.lifecycle = ANY($1)
			AND project.flags = 0
		ORDER BY project.date_approved ASC
		`,
		models.VisibleProjectLifecycles,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch projects"))
	}
	allProjectsSlice := allProjects.ToSlice()
	c.Perf.EndBlock()

	numPages := int(math.Ceil(float64(len(allProjectsSlice)) / projectsPerPage))

	if page > numPages {
		return c.Redirect(hmnurl.BuildProjectIndex(numPages), http.StatusSeeOther)
	}

	pagination := templates.Pagination{
		Current: page,
		Total:   numPages,

		FirstUrl:    hmnurl.BuildProjectIndex(1),
		LastUrl:     hmnurl.BuildProjectIndex(numPages),
		NextUrl:     hmnurl.BuildProjectIndex(utils.IntClamp(1, page+1, numPages)),
		PreviousUrl: hmnurl.BuildProjectIndex(utils.IntClamp(1, page-1, numPages)),
	}

	var userApprovedProjects []templates.Project
	var userPendingProject *templates.Project
	userPendingProjectUnderReview := false
	if c.CurrentUser != nil {
		c.Perf.StartBlock("SQL", "fetching user projects")
		type UserProjectQuery struct {
			Project models.Project `db:"project"`
		}
		userProjectsResult, err := db.Query(c.Context(), c.Conn, UserProjectQuery{},
			`
			SELECT $columns
			FROM
				handmade_project AS project
				INNER JOIN handmade_user_projects AS uproj ON uproj.project_id = project.id
			WHERE
				uproj.user_id = $1
			`,
			c.CurrentUser.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user projects"))
		}
		for _, project := range userProjectsResult.ToSlice() {
			p := project.(*UserProjectQuery).Project
			if p.Lifecycle == models.ProjectLifecycleUnapproved || p.Lifecycle == models.ProjectLifecycleApprovalRequired {
				if userPendingProject == nil {
					// NOTE(asaf): Technically a user could have more than one pending project.
					//			   For example, if they created one project themselves and were added as an additional owner to another user's project.
					//             So we'll just take the first one. I don't think it matters. I guess it especially won't matter after Projects 2.0.
					tmplProject := templates.ProjectToTemplate(&p, c.Theme)
					userPendingProject = &tmplProject
					userPendingProjectUnderReview = (p.Lifecycle == models.ProjectLifecycleApprovalRequired)
				}
			} else {
				userApprovedProjects = append(userApprovedProjects, templates.ProjectToTemplate(&p, c.Theme))
			}
		}
		c.Perf.EndBlock()
	}

	c.Perf.StartBlock("PROJECTS", "Grouping and sorting")
	var handmadeHero *templates.Project
	var featuredProjects []templates.Project
	var recentProjects []templates.Project
	var restProjects []templates.Project
	now := time.Now()
	for _, p := range allProjectsSlice {
		project := &p.(*projectResult).Project
		templateProject := templates.ProjectToTemplate(project, c.Theme)
		if project.Slug == "hero" {
			// NOTE(asaf): Handmade Hero gets special treatment. Must always be first in the list.
			handmadeHero = &templateProject
			continue
		}
		if project.Featured {
			featuredProjects = append(featuredProjects, templateProject)
		} else if now.Sub(project.AllLastUpdated).Seconds() < models.RecentProjectUpdateTimespanSec {
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

	baseData := getBaseData(c)
	baseData.Title = "Project List"
	var res ResponseData
	res.MustWriteTemplate("project_index.html", ProjectTemplateData{
		BaseData: baseData,

		Pagination:       pagination,
		CarouselProjects: carouselProjects,
		Projects:         pageProjects,

		UserPendingProjectUnderReview: userPendingProjectUnderReview,
		UserPendingProject:            userPendingProject,
		UserApprovedProjects:          userApprovedProjects,

		ProjectAtomFeedUrl: hmnurl.BuildAtomFeedForProjects(),
		ManifestoUrl:       hmnurl.BuildManifesto(),
		NewProjectUrl:      hmnurl.BuildProjectNew(),
		RegisterUrl:        hmnurl.BuildRegister(),
		LoginUrl:           hmnurl.BuildLoginPage(c.FullUrl()),
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
				if errors.Is(err, db.ErrNoMatchingRows) {
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

	owners, err := FetchProjectOwners(c, project.ID)
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
		if project.Flags == 0 {
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

	projectHomepageData.BaseData = getBaseData(c)
	if canEdit {
		projectHomepageData.BaseData.Header.EditUrl = hmnurl.BuildProjectEdit(project.Slug, "")
	}
	projectHomepageData.Project = templates.ProjectToTemplate(project, c.Theme)
	for _, owner := range owners {
		projectHomepageData.Owners = append(projectHomepageData.Owners, templates.UserToTemplate(owner, c.Theme))
	}

	if project.Flags == 1 {
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

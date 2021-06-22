package website

import (
	"math"
	"math/rand"
	"net/http"
	"strconv"
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
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch projects"))
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
				INNER JOIN handmade_project_groups AS project_groups ON project_groups.project_id = project.id
				INNER JOIN auth_user_groups AS user_groups ON user_groups.group_id = project_groups.group_id
			WHERE
				user_groups.user_id = $1
			`,
			c.CurrentUser.ID,
		)
		if err != nil {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user projects"))
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
	err = res.WriteTemplate("project_index.html", ProjectTemplateData{
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
	if err != nil {
		panic(err)
	}
	return res
}

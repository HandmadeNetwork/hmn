package website

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"math/rand"
	"net/http"
	"path"
	"sort"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/twitch"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const maxPersonalProjects = 10
const maxProjectOwners = 5

func ProjectCSS(c *RequestContext) ResponseData {
	color := c.URL().Query().Get("color")
	if color == "" {
		return c.ErrorResponse(http.StatusBadRequest, NewSafeError(nil, "You must provide a 'color' parameter.\n"))
	}

	baseData := getBaseData(c, "", nil)

	templateData := struct {
		templates.BaseData
		Color string
	}{
		BaseData: baseData,
		Color:    color,
	}

	var res ResponseData
	res.Header().Add("Content-Type", "text/css")
	err := res.WriteTemplate("project.css", templateData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to generate project CSS"))
	}

	return res
}

type ProjectTemplateData struct {
	templates.BaseData

	AllProjects bool

	// Stuff for all projects
	OfficialProjects        []templates.Project
	OfficialProjectsLink    string
	PersonalProjects        []templates.Project
	PersonalProjectsLink    string
	CurrentJamProjects      []templates.Project
	CurrentJamProjectsLink  string
	CurrentJamLink          string
	CurrentJamSlug          string
	PreviousJamProjects     []templates.Project
	PreviousJamProjectsLink string
	PreviousJamLink         string
	PreviousJamSlug         string

	// Stuff for pages of projects only
	Category     string
	Pagination   templates.Pagination
	PageProjects []templates.Project
	PageJamLink  string

	// Stuff for both
	CreateProjectLink string
}

func ProjectIndex(c *RequestContext) ResponseData {
	cat := c.PathParams["category"]
	pageStr := c.PathParams["page"]

	currentJam := hmndata.CurrentJam()
	previousJam := hmndata.PreviousJam()

	if cat == "" && pageStr == "" {
		const projectsPerSection = 8

		officialProjects, err := getShuffledOfficialProjects(c)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, err)
		}

		personalProjects, err := getPersonalProjects(c, "")
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, err)
		}

		var currentJamProjects []templates.Project
		if currentJam != nil {
			var err error
			currentJamProjects, err = getPersonalProjects(c, currentJam.Slug)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, err)
			}
		}

		previousJamProjects, err := getPersonalProjects(c, previousJam.Slug)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, err)
		}

		baseData := getBaseDataAutocrumb(c, "Projects")
		tmpl := ProjectTemplateData{
			BaseData: baseData,

			AllProjects: true,

			OfficialProjects:     officialProjects[:utils.Min(len(officialProjects), projectsPerSection)],
			OfficialProjectsLink: hmnurl.BuildProjectIndex(1, "official"),
			PersonalProjects:     personalProjects[:utils.Min(len(personalProjects), projectsPerSection)],
			PersonalProjectsLink: hmnurl.BuildProjectIndex(1, "personal"),
			// Current jam stuff set later
			PreviousJamProjects:     previousJamProjects[:utils.Min(len(previousJamProjects), projectsPerSection)],
			PreviousJamProjectsLink: hmnurl.BuildProjectIndex(1, previousJam.UrlSlug),
			PreviousJamLink:         hmnurl.BuildJamIndexAny(previousJam.UrlSlug),
			PreviousJamSlug:         previousJam.UrlSlug,

			CreateProjectLink: hmnurl.BuildProjectNew(),
		}

		if currentJam != nil {
			tmpl.CurrentJamProjects = currentJamProjects[:utils.Min(len(currentJamProjects), projectsPerSection)]
			tmpl.CurrentJamProjectsLink = hmnurl.BuildProjectIndex(1, currentJam.UrlSlug)
			tmpl.CurrentJamLink = hmnurl.BuildJamIndexAny(currentJam.UrlSlug)
			tmpl.CurrentJamSlug = currentJam.UrlSlug
		}

		var res ResponseData
		res.MustWriteTemplate("project_index.html", tmpl, c.Perf)
		return res
	} else {
		const projectsPerPage = 20

		var breadcrumb templates.Breadcrumb
		breadcrumbUrl := hmnurl.BuildProjectIndex(1, cat)

		var projects []templates.Project
		var jamLink string

		var err error
		switch cat {
		case hmndata.WRJ2022.UrlSlug:
			projects, err = getPersonalProjects(c, hmndata.WRJ2022.Slug)
			breadcrumb = templates.Breadcrumb{Name: "Wheel Reinvention Jam 2022"}
			jamLink = hmnurl.BuildJamIndex2022()
		case hmndata.VJ2023.UrlSlug:
			projects, err = getPersonalProjects(c, hmndata.VJ2023.Slug)
			breadcrumb = templates.Breadcrumb{"2023 Visibility Jam", breadcrumbUrl}
			jamLink = hmnurl.BuildJamIndex2023_Visibility()
		case hmndata.WRJ2023.UrlSlug:
			projects, err = getPersonalProjects(c, hmndata.WRJ2023.Slug)
			breadcrumb = templates.Breadcrumb{"Wheel Reinvention Jam 2023", breadcrumbUrl}
			jamLink = hmnurl.BuildJamIndex2023()
		case "personal":
			projects, err = getPersonalProjects(c, "")
			breadcrumb = templates.Breadcrumb{"Personal Projects", breadcrumbUrl}
		case "official":
			projects, err = getShuffledOfficialProjects(c)
			breadcrumb = templates.Breadcrumb{"Official Projects", breadcrumbUrl}
		default:
			return c.Redirect(hmnurl.BuildProjectIndex(1, ""), http.StatusSeeOther)
		}
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, err)
		}

		page, numPages, ok := getPageInfo(pageStr, len(projects), projectsPerPage)
		if !ok {
			return c.Redirect(hmnurl.BuildProjectIndex(1, cat), http.StatusSeeOther)
		}
		pagination := templates.Pagination{
			Current: page,
			Total:   numPages,

			FirstUrl:    hmnurl.BuildProjectIndex(1, cat),
			LastUrl:     hmnurl.BuildProjectIndex(numPages, cat),
			NextUrl:     hmnurl.BuildProjectIndex(utils.Clamp(1, page+1, numPages), cat),
			PreviousUrl: hmnurl.BuildProjectIndex(utils.Clamp(1, page-1, numPages), cat),
		}

		firstProjectIndex := (page - 1) * projectsPerPage
		endIndex := utils.Min(firstProjectIndex+projectsPerPage, len(projects))
		pageProjects := projects[firstProjectIndex:endIndex]

		baseData := getBaseData(c, "Projects", []templates.Breadcrumb{
			{"Projects", hmnurl.BuildProjectIndex(1, "")},
			breadcrumb,
		})
		var res ResponseData
		res.MustWriteTemplate("project_index.html", ProjectTemplateData{
			BaseData: baseData,

			AllProjects: false,

			Category:          cat,
			Pagination:        pagination,
			PageProjects:      pageProjects,
			CreateProjectLink: hmnurl.BuildProjectNew(),
			PageJamLink:       jamLink,
		}, c.Perf)
		return res
	}
}

func getShuffledOfficialProjects(c *RequestContext) ([]templates.Project, error) {
	official, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		Types: hmndata.OfficialProjects,
	})
	if err != nil {
		return nil, oops.New(err, "failed to fetch projects")
	}

	c.Perf.StartBlock("PROJECTS", "Grouping and sorting")
	var handmadeHero *templates.Project
	var featuredProjects []templates.Project
	var recentProjects []templates.Project
	var restProjects []templates.Project
	now := time.Now()
	for _, p := range official {
		templateProject := templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage())

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

	officialProjects := make([]templates.Project, 0, len(featuredProjects)+len(recentProjects)+len(restProjects))
	officialProjects = append(officialProjects, featuredProjects...)
	officialProjects = append(officialProjects, recentProjects...)
	officialProjects = append(officialProjects, restProjects...)

	c.Perf.EndBlock()

	return officialProjects, nil
}

func getPersonalProjects(c *RequestContext, jamSlug string) ([]templates.Project, error) {
	var slugs []string
	if jamSlug != "" {
		slugs = []string{jamSlug}
	}

	projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		Types:    hmndata.PersonalProjects,
		JamSlugs: slugs,
	})
	if err != nil {
		return nil, oops.New(err, "failed to fetch personal projects")
	}

	sort.Slice(projects, func(i, j int) bool {
		p1 := projects[i].Project
		p2 := projects[j].Project
		return p2.AllLastUpdated.Before(p1.AllLastUpdated) // sort backwards - recent first
	})

	var personalProjects []templates.Project
	for _, p := range projects {
		templateProject := templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage())
		personalProjects = append(personalProjects, templateProject)
	}

	return personalProjects, nil
}

func jamLink(jamSlug string) string {
	switch jamSlug {
	case hmndata.WRJ2021.Slug:
		return hmnurl.BuildJamIndex2021()
	case hmndata.WRJ2022.Slug:
		return hmnurl.BuildJamIndex2022()
	case hmndata.WRJ2023.Slug:
		return hmnurl.BuildJamIndex2023()
	case hmndata.VJ2023.Slug:
		return hmnurl.BuildJamIndex2023_Visibility()
	default:
		return ""
	}
}

type ProjectHomepageData struct {
	templates.BaseData
	Project        templates.Project
	Owners         []templates.User
	Screenshots    []string
	ProjectLinks   []templates.Link
	Licenses       []templates.Link
	RecentActivity []templates.TimelineItem
	SnippetEdit    templates.SnippetEdit

	FollowUrl string
	Following bool
}

func ProjectHomepage(c *RequestContext) ResponseData {
	maxRecentActivity := 100

	if c.CurrentProject == nil {
		return FourOhFour(c)
	}

	// There are no further permission checks to do, because permissions are
	// checked whatever way we fetch the project.

	owners, err := hmndata.FetchProjectOwners(c, c.Conn, c.CurrentProject.ID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	c.Perf.StartBlock("SQL", "Fetching screenshots")
	screenshotFilenames, err := db.QueryScalar[string](c, c.Conn,
		`
		SELECT screenshot.file
		FROM
			image_file AS screenshot
			INNER JOIN project_screenshot ON screenshot.id = project_screenshot.imagefile_id
		WHERE
			project_screenshot.project_id = $1
		`,
		c.CurrentProject.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch screenshots for project"))
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetching project links")
	projectLinks, err := db.Query[models.Link](c, c.Conn,
		`
		SELECT $columns
		FROM
			link as link
		WHERE
			link.project_id = $1
		ORDER BY link.ordering ASC
		`,
		c.CurrentProject.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project links"))
	}
	c.Perf.EndBlock()

	type ProjectHomepageData struct {
		templates.BaseData
		Project                  templates.Project
		Owners                   []templates.User
		Screenshots              []string
		NamedLinks, UnnamedLinks []templates.Link
		RecentActivity           []templates.TimelineItem
		SnippetEdit              templates.SnippetEdit

		FollowUrl string
		Following bool
	}

	var templateData ProjectHomepageData

	templateData.BaseData = getBaseData(c, c.CurrentProject.Name, nil)
	templateData.BaseData.OpenGraphItems = append(templateData.BaseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    c.CurrentProject.Blurb,
	})

	p, err := hmndata.FetchProject(c, c.Conn, c.CurrentUser, c.CurrentProject.ID, hmndata.ProjectsQuery{
		Lifecycles:    models.AllProjectLifecycles,
		IncludeHidden: true,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project details"))
	}
	templateData.Project = templates.ProjectAndStuffToTemplate(&p, c.UrlContext.BuildHomepage())
	for _, owner := range owners {
		templateData.Owners = append(templateData.Owners, templates.UserToTemplate(owner))
	}

	if c.CurrentProject.Hidden {
		templateData.BaseData.AddImmediateNotice(
			"hidden",
			"NOTICE: This project is hidden. It is currently visible only to owners and site admins.",
		)
	}

	if c.CurrentProject.Lifecycle != models.ProjectLifecycleActive {
		switch c.CurrentProject.Lifecycle {
		case models.ProjectLifecycleUnapproved:
			templateData.BaseData.AddImmediateNotice(
				"unapproved",
				fmt.Sprintf(
					"NOTICE: This project has not yet been submitted for approval. It is only visible to owners. Please <a href=\"%s\">submit it for approval</a> when the project content is ready for review.",
					c.UrlContext.BuildProjectEdit("submit"),
				),
			)
		case models.ProjectLifecycleApprovalRequired:
			templateData.BaseData.AddImmediateNotice(
				"unapproved",
				"NOTICE: This project is awaiting approval. It is only visible to owners and site admins.",
			)
		case models.ProjectLifecycleHiatus:
			templateData.BaseData.AddImmediateNotice(
				"hiatus",
				"NOTICE: This project is on hiatus and may not update for a while.",
			)
		case models.ProjectLifecycleDead:
			templateData.BaseData.AddImmediateNotice(
				"dead",
				"NOTICE: This project is has been marked dead and is only visible to owners and site admins.",
			)
		case models.ProjectLifecycleLTSRequired:
			templateData.BaseData.AddImmediateNotice(
				"lts-reqd",
				"NOTICE: This project is awaiting approval for maintenance-mode status.",
			)
		}
	}

	for _, screenshotFilename := range screenshotFilenames {
		templateData.Screenshots = append(templateData.Screenshots, hmnurl.BuildUserFile(screenshotFilename))
	}

	for _, link := range templates.LinksToTemplate(projectLinks) {
		if link.Name != "" {
			templateData.NamedLinks = append(templateData.NamedLinks, link)
		} else {
			templateData.UnnamedLinks = append(templateData.UnnamedLinks, link)
		}
	}

	templateData.RecentActivity, err = FetchTimeline(c, c.Conn, c.CurrentUser, TimelineQuery{
		ProjectIDs: []int{c.CurrentProject.ID},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	templateData.RecentActivity = utils.ClampSlice(templateData.RecentActivity, maxRecentActivity)

	followUrl := ""
	following := false
	if c.CurrentUser != nil {
		userProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user projects"))
		}
		templateProjects := make([]templates.Project, 0, len(userProjects))
		templateProjects = append(templateProjects, templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage()))
		for _, p := range userProjects {
			if p.Project.ID == c.CurrentProject.ID {
				continue
			}
			templateProject := templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage())
			templateProjects = append(templateProjects, templateProject)
		}
		templateData.SnippetEdit = templates.SnippetEdit{
			AvailableProjectsJSON: templates.SnippetEditProjectsToJSON(templateProjects),
			SubmitUrl:             hmnurl.BuildSnippetSubmit(),
			AssetMaxSize:          AssetMaxSize(c.CurrentUser),
		}

		followUrl = hmnurl.BuildFollowProject()
		following, err = db.QueryOneScalar[bool](c, c.Conn, `
			SELECT COUNT(*) > 0
			FROM follower
			WHERE user_id = $1 AND following_project_id = $2
		`, c.CurrentUser.ID, c.CurrentProject.ID)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch following status"))
		}
	}
	templateData.FollowUrl = followUrl
	templateData.Following = following

	var res ResponseData
	err = res.WriteTemplate("project_homepage.html", templateData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render project homepage template"))
	}
	return res
}

var ProjectLogoMaxFileSize = 2 * 1024 * 1024
var ProjectHeaderMaxFileSize = 2 * 1024 * 1024 // TODO(ben): Pick a real limit

type ProjectEditData struct {
	templates.BaseData

	Editing         bool
	ProjectSettings templates.ProjectSettings
	MaxOwners       int

	APICheckUsernameUrl                string
	LogoMaxFileSize, HeaderMaxFileSize int

	TextEditor templates.TextEditor
}

func ProjectNew(c *RequestContext) ResponseData {
	numProjects, err := hmndata.CountProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		OwnerIDs: []int{c.CurrentUser.ID},
		Types:    hmndata.PersonalProjects,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to check number of personal projects"))
	}
	if numProjects >= maxPersonalProjects {
		return c.RejectRequest(fmt.Sprintf("You have already reached the maximum of %d personal projects.", maxPersonalProjects))
	}

	var project templates.ProjectSettings
	project.Owners = append(project.Owners, templates.UserToTemplate(c.CurrentUser))
	project.Personal = true

	var currentJam *hmndata.Jam
	if c.Req.URL.Query().Has("jam") {
		currentJam = hmndata.CurrentJam()
		if currentJam != nil {
			project.JamParticipation = []templates.ProjectJamParticipation{
				templates.ProjectJamParticipation{
					JamName:       currentJam.Name,
					JamSlug:       currentJam.Slug,
					Participating: true,
				},
			}
		}
	}

	var res ResponseData
	res.MustWriteTemplate("project_edit.html", ProjectEditData{
		BaseData:        getBaseDataAutocrumb(c, "New Project"),
		Editing:         false,
		ProjectSettings: project,
		MaxOwners:       maxProjectOwners,

		APICheckUsernameUrl: hmnurl.BuildAPICheckUsername(),
		LogoMaxFileSize:     ProjectLogoMaxFileSize,
		HeaderMaxFileSize:   ProjectHeaderMaxFileSize,

		TextEditor: templates.TextEditor{
			MaxFileSize: AssetMaxSize(c.CurrentUser),
			UploadUrl:   c.UrlContext.BuildAssetUpload(),
		},
	}, c.Perf)
	return res
}

func ProjectNewSubmit(c *RequestContext) ResponseData {
	formResult := ParseProjectEditForm(c)
	if formResult.Error != nil {
		return c.ErrorResponse(http.StatusInternalServerError, formResult.Error)
	}
	if len(formResult.RejectionReason) != 0 {
		return c.RejectRequest(formResult.RejectionReason)
	}

	tx, err := c.Conn.Begin(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to start db transaction"))
	}
	defer tx.Rollback(c)

	numProjects, err := hmndata.CountProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		OwnerIDs: []int{c.CurrentUser.ID},
		Types:    hmndata.PersonalProjects,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to check number of personal projects"))
	}
	if numProjects >= maxPersonalProjects {
		return c.RejectRequest(fmt.Sprintf("You have already reached the maximum of %d personal projects.", maxPersonalProjects))
	}

	var projectId int
	err = tx.QueryRow(c,
		`
		INSERT INTO project
			(name, blurb, description, descparsed, lifecycle, date_created, all_last_updated)
		VALUES
			($1,   $2,    $3,          $4,         $5,        $6,           $6)
		RETURNING id
		`,
		"",
		"",
		"",
		"",
		models.ProjectLifecycleUnapproved,
		time.Now(), // NOTE(asaf): Using this param twice.
	).Scan(&projectId)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to insert new project"))
	}

	formResult.Payload.ProjectID = projectId

	err = updateProject(c, tx, c.CurrentUser, &formResult.Payload)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	tx.Commit(c)

	urlContext := &hmnurl.UrlContext{
		PersonalProject: true,
		ProjectID:       projectId,
		ProjectName:     formResult.Payload.Name,
	}

	return c.Redirect(urlContext.BuildHomepage(), http.StatusSeeOther)
}

func ProjectEdit(c *RequestContext) ResponseData {
	if !c.CurrentUserCanEditCurrentProject {
		return FourOhFour(c)
	}

	p, err := hmndata.FetchProject(
		c, c.Conn,
		c.CurrentUser, c.CurrentProject.ID,
		hmndata.ProjectsQuery{
			Lifecycles:    models.AllProjectLifecycles,
			IncludeHidden: true,
		},
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	c.Perf.StartBlock("SQL", "Fetching project links")
	projectLinks, err := db.Query[models.Link](c, c.Conn,
		`
		SELECT $columns
		FROM
			link as link
		WHERE
			link.project_id = $1
		ORDER BY link.ordering ASC
		`,
		p.Project.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project links"))
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetching project jams")
	projectJams, err := hmndata.FetchJamsForProject(c, c.Conn, c.CurrentUser, p.Project.ID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jams for project"))
	}
	c.Perf.EndBlock()

	projectSettings := templates.ProjectToProjectSettings(
		&p.Project,
		p.Owners,
		p.TagText(),
		p.LogoLightAsset, p.LogoDarkAsset, p.HeaderImage,
	)

	projectSettings.LinksJSON = string(utils.Must1(json.Marshal(templates.LinksToTemplate(projectLinks))))

	projectSettings.JamParticipation = make([]templates.ProjectJamParticipation, 0, len(projectJams))
	for _, jam := range projectJams {
		projectSettings.JamParticipation = append(projectSettings.JamParticipation, templates.ProjectJamParticipation{
			JamName:       jam.JamName,
			JamSlug:       jam.JamSlug,
			Participating: jam.Participating,
		})
	}

	var res ResponseData
	res.MustWriteTemplate("project_edit.html", ProjectEditData{
		BaseData:        getBaseDataAutocrumb(c, "Edit Project"),
		Editing:         true,
		ProjectSettings: projectSettings,
		MaxOwners:       maxProjectOwners,

		APICheckUsernameUrl: hmnurl.BuildAPICheckUsername(),
		LogoMaxFileSize:     ProjectLogoMaxFileSize,
		HeaderMaxFileSize:   ProjectHeaderMaxFileSize,

		TextEditor: templates.TextEditor{
			MaxFileSize: AssetMaxSize(c.CurrentUser),
			UploadUrl:   c.UrlContext.BuildAssetUpload(),
		},
	}, c.Perf)
	return res
}

func ProjectEditSubmit(c *RequestContext) ResponseData {
	if !c.CurrentUserCanEditCurrentProject {
		return FourOhFour(c)
	}
	formResult := ParseProjectEditForm(c)
	if formResult.Error != nil {
		return c.ErrorResponse(http.StatusInternalServerError, formResult.Error)
	}
	if len(formResult.RejectionReason) != 0 {
		return c.RejectRequest(formResult.RejectionReason)
	}

	tx, err := c.Conn.Begin(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to start db transaction"))
	}
	defer tx.Rollback(c)

	formResult.Payload.ProjectID = c.CurrentProject.ID

	err = updateProject(c, tx, c.CurrentUser, &formResult.Payload)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	tx.Commit(c)

	urlContext := &hmnurl.UrlContext{
		PersonalProject: formResult.Payload.Personal,
		ProjectSlug:     formResult.Payload.Slug,
		ProjectID:       formResult.Payload.ProjectID,
		ProjectName:     formResult.Payload.Name,
	}

	return c.Redirect(urlContext.BuildHomepage(), http.StatusSeeOther)
}

type ProjectPayload struct {
	ProjectID             int
	Name                  string
	Blurb                 string
	Links                 []ParsedLink
	Description           string
	ParsedDescription     string
	Lifecycle             models.ProjectLifecycle
	Hidden                bool
	OwnerUsernames        []string
	LightLogo             FormImage
	DarkLogo              FormImage
	HeaderImage           FormImage
	Tag                   string
	JamParticipationSlugs []string

	Slug     string
	Featured bool
	Personal bool
}

type ProjectEditFormResult struct {
	Payload         ProjectPayload
	RejectionReason string
	Error           error
}

func ParseProjectEditForm(c *RequestContext) ProjectEditFormResult {
	var res ProjectEditFormResult
	maxBodySize := int64(ProjectLogoMaxFileSize*2 + 1024*1024)
	c.Req.Body = http.MaxBytesReader(c.Res, c.Req.Body, maxBodySize)
	err := c.Req.ParseMultipartForm(maxBodySize)
	if err != nil {
		// NOTE(asaf): The error for exceeding the max filesize doesn't have a special type, so we can't easily detect it here.
		res.Error = oops.New(err, "failed to parse form")
		return res
	}

	projectName := strings.TrimSpace(c.Req.Form.Get("project_name"))
	if len(projectName) == 0 {
		res.RejectionReason = "Project name is empty"
		return res
	}

	shortDesc := strings.TrimSpace(c.Req.Form.Get("shortdesc"))
	if len(shortDesc) == 0 {
		res.RejectionReason = "Projects must have a short description"
		return res
	}
	links := ParseLinks(c.Req.Form.Get("links"))
	description := c.Req.Form.Get("full_description")
	parsedDescription := parsing.ParseMarkdown(description, parsing.ForumRealMarkdown)

	lifecycleStr := c.Req.Form.Get("lifecycle")
	lifecycle, found := templates.ProjectLifecycleFromValue(lifecycleStr)
	if !found {
		res.RejectionReason = "Project status is invalid"
		return res
	}

	tag := c.Req.Form.Get("tag")
	if !models.ValidateTagText(tag) {
		res.RejectionReason = "Project tag is invalid"
		return res
	}

	hiddenStr := c.Req.Form.Get("hidden")
	hidden := len(hiddenStr) > 0

	lightLogo, err := GetFormImage(c, "light_logo")
	if err != nil {
		res.Error = oops.New(err, "Failed to read image from form")
		return res
	}
	darkLogo, err := GetFormImage(c, "dark_logo")
	if err != nil {
		res.Error = oops.New(err, "Failed to read image from form")
		return res
	}
	headerImage, err := GetFormImage(c, "header_image")
	if err != nil {
		res.Error = oops.New(err, "Failed to read image from form")
		return res
	}

	owners := c.Req.Form["owners"]
	if len(owners) > maxProjectOwners {
		res.RejectionReason = fmt.Sprintf("Projects can have at most %d owners", maxProjectOwners)
		return res
	}

	slug := strings.TrimSpace(c.Req.Form.Get("slug"))
	officialStr := c.Req.Form.Get("official")
	official := len(officialStr) > 0
	featuredStr := c.Req.Form.Get("featured")
	featured := len(featuredStr) > 0

	if official && len(slug) == 0 {
		res.RejectionReason = "Official projects must have a slug"
		return res
	}

	jamParticipationSlugs := c.Req.Form["jam_participation"]

	res.Payload = ProjectPayload{
		Name:                  projectName,
		Blurb:                 shortDesc,
		Links:                 links,
		Description:           description,
		ParsedDescription:     parsedDescription,
		Lifecycle:             lifecycle,
		Hidden:                hidden,
		OwnerUsernames:        owners,
		LightLogo:             lightLogo,
		DarkLogo:              darkLogo,
		HeaderImage:           headerImage,
		Tag:                   tag,
		JamParticipationSlugs: jamParticipationSlugs,
		Slug:                  slug,
		Personal:              !official,
		Featured:              featured,
	}

	return res
}

func updateProject(ctx context.Context, tx pgx.Tx, user *models.User, payload *ProjectPayload) error {
	var lightLogoUUID *uuid.UUID
	if payload.LightLogo.Exists {
		lightLogo := &payload.LightLogo
		lightLogoAsset, err := assets.Create(ctx, tx, assets.CreateInput{
			Content:     lightLogo.Content,
			Filename:    lightLogo.Filename,
			ContentType: lightLogo.Mime,
			UploaderID:  &user.ID,
			Width:       lightLogo.Width,
			Height:      lightLogo.Height,
		})
		if err != nil {
			return oops.New(err, "Failed to save asset")
		}
		lightLogoUUID = &lightLogoAsset.ID
	}

	var darkLogoUUID *uuid.UUID
	if payload.DarkLogo.Exists {
		darkLogo := &payload.DarkLogo
		darkLogoAsset, err := assets.Create(ctx, tx, assets.CreateInput{
			Content:     darkLogo.Content,
			Filename:    darkLogo.Filename,
			ContentType: darkLogo.Mime,
			UploaderID:  &user.ID,
			Width:       darkLogo.Width,
			Height:      darkLogo.Height,
		})
		if err != nil {
			return oops.New(err, "Failed to save asset")
		}
		darkLogoUUID = &darkLogoAsset.ID
	}

	var headerImageUUID *uuid.UUID
	if payload.HeaderImage.Exists {
		headerImage := &payload.HeaderImage
		headerImageAsset, err := assets.Create(ctx, tx, assets.CreateInput{
			Content:     headerImage.Content,
			Filename:    headerImage.Filename,
			ContentType: headerImage.Mime,
			UploaderID:  &user.ID,
			Width:       headerImage.Width,
			Height:      headerImage.Height,
		})
		if err != nil {
			return oops.New(err, "Failed to save asset")
		}
		headerImageUUID = &headerImageAsset.ID
	}

	hasSelf := false
	selfUsername := strings.ToLower(user.Username)
	for i, _ := range payload.OwnerUsernames {
		payload.OwnerUsernames[i] = strings.ToLower(payload.OwnerUsernames[i])
		if payload.OwnerUsernames[i] == selfUsername {
			hasSelf = true
		}
	}

	if !hasSelf && !user.IsStaff {
		payload.OwnerUsernames = append(payload.OwnerUsernames, selfUsername)
	}

	_, err := tx.Exec(ctx,
		`
		UPDATE project SET
			name = $2,
			blurb = $3,
			description = $4,
			descparsed = $5,
			lifecycle = $6
		WHERE id = $1
		`,
		payload.ProjectID,
		payload.Name,
		payload.Blurb,
		payload.Description,
		payload.ParsedDescription,
		payload.Lifecycle,
	)
	if err != nil {
		return oops.New(err, "Failed to update project")
	}

	_, err = hmndata.SetProjectTag(ctx, tx, user, payload.ProjectID, payload.Tag)
	if err != nil {
		return err
	}

	if user.IsStaff {
		_, err = tx.Exec(ctx,
			`
			UPDATE project SET
				slug = $2,
				featured = $3,
				personal = $4,
				hidden = $5
			WHERE
				id = $1
			`,
			payload.ProjectID,
			payload.Slug,
			payload.Featured,
			payload.Personal,
			payload.Hidden,
		)
		if err != nil {
			return oops.New(err, "Failed to update project with admin fields")
		}
	}

	if payload.LightLogo.Exists || payload.LightLogo.Remove {
		_, err = tx.Exec(ctx,
			`
			UPDATE project
			SET
				logolight_asset_id = $2
			WHERE
				id = $1
			`,
			payload.ProjectID,
			lightLogoUUID,
		)
		if err != nil {
			return oops.New(err, "Failed to update project's light logo")
		}
	}

	if payload.DarkLogo.Exists || payload.DarkLogo.Remove {
		_, err = tx.Exec(ctx,
			`
			UPDATE project
			SET
				logodark_asset_id = $2
			WHERE
				id = $1
			`,
			payload.ProjectID,
			darkLogoUUID,
		)
		if err != nil {
			return oops.New(err, "Failed to update project's dark logo")
		}
	}

	if payload.HeaderImage.Exists || payload.HeaderImage.Remove {
		_, err = tx.Exec(ctx,
			`
			UPDATE project
			SET
				header_asset_id = $2
			WHERE
				id = $1
			`,
			payload.ProjectID,
			headerImageUUID,
		)
		if err != nil {
			return oops.New(err, "Failed to update project's header image")
		}
	}

	owners, err := db.Query[models.User](ctx, tx,
		`
		SELECT $columns
		FROM hmn_user
		WHERE LOWER(username) = ANY ($1)
		`,
		payload.OwnerUsernames,
	)
	if err != nil {
		return oops.New(err, "Failed to query users")
	}

	_, err = tx.Exec(ctx,
		`
		DELETE FROM user_project
		WHERE project_id = $1
		`,
		payload.ProjectID,
	)
	if err != nil {
		return oops.New(err, "Failed to delete project owners")
	}

	for _, owner := range owners {
		_, err = tx.Exec(ctx,
			`
			INSERT INTO user_project
				(user_id, project_id)
			VALUES
				($1,      $2)
			`,
			owner.ID,
			payload.ProjectID,
		)
		if err != nil {
			return oops.New(err, "Failed to insert project owner")
		}
	}

	twitchLoginsPreChange, preErr := hmndata.FetchTwitchLoginsForUserOrProject(ctx, tx, nil, &payload.ProjectID)
	_, err = tx.Exec(ctx, `DELETE FROM link WHERE project_id = $1`, payload.ProjectID)
	if err != nil {
		return oops.New(err, "Failed to delete project links")
	}
	for i, link := range payload.Links {
		_, err = tx.Exec(ctx,
			`
			INSERT INTO link (name, url, ordering, project_id)
			VALUES ($1, $2, $3, $4)
			`,
			link.Name,
			link.Url,
			i,
			payload.ProjectID,
		)
		if err != nil {
			return oops.New(err, "Failed to insert new project link")
		}
	}
	twitchLoginsPostChange, postErr := hmndata.FetchTwitchLoginsForUserOrProject(ctx, tx, nil, &payload.ProjectID)
	if preErr == nil && postErr == nil {
		twitch.UserOrProjectLinksUpdated(twitchLoginsPreChange, twitchLoginsPostChange)
	}

	// NOTE(asaf): Regular users can only edit the jam participation status of the current jam or
	//             jams the project was previously a part of.
	var possibleJamSlugs []string
	if user.IsStaff {
		possibleJamSlugs = make([]string, 0, len(hmndata.AllJams))
		for _, jam := range hmndata.AllJams {
			possibleJamSlugs = append(possibleJamSlugs, jam.Slug)
		}
	} else {
		possibleJamSlugs, err = db.QueryScalar[string](ctx, tx,
			`
			SELECT jam_slug
			FROM jam_project
			WHERE project_id = $1
			`,
			payload.ProjectID,
		)
		if err != nil {
			return oops.New(err, "Failed to fetch jam participation for project")
		}
		currentJam := hmndata.CurrentJam()
		if currentJam != nil {
			possibleJamSlugs = append(possibleJamSlugs, currentJam.Slug)
		}
	}

	_, err = tx.Exec(ctx,
		`
		UPDATE jam_project
		SET participating = FALSE
		WHERE project_id = $1
		`,
		payload.ProjectID,
	)
	if err != nil {
		return oops.New(err, "Failed to remove jam participation for project")
	}

	for _, jamSlug := range payload.JamParticipationSlugs {
		found := false
		for _, possibleSlug := range possibleJamSlugs {
			if possibleSlug == jamSlug {
				found = true
				break
			}
		}
		if found {
			_, err = tx.Exec(ctx,
				`
				INSERT INTO jam_project (project_id, jam_slug, participating)
				VALUES ($1, $2, $3)
				ON CONFLICT (project_id, jam_slug) DO UPDATE SET
					participating = EXCLUDED.participating
				`,
				payload.ProjectID,
				jamSlug,
				true,
			)
			if err != nil {
				return oops.New(err, "Failed to insert/update jam participation for project")
			}
		}
	}

	return nil
}

type FormImage struct {
	Exists   bool
	Remove   bool
	Filename string
	Mime     string
	Content  []byte
	Width    int
	Height   int
	Size     int64
}

// NOTE(asaf): This assumes that you already called ParseMultipartForm (which is why there's no size limit here).
func GetFormImage(c *RequestContext, fieldName string) (FormImage, error) {
	var res FormImage
	res.Exists = false

	removeStr := c.Req.Form.Get("remove_" + fieldName)
	res.Remove = (removeStr == "true")
	img, header, err := c.Req.FormFile(fieldName)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return res, nil
		} else {
			return FormImage{}, err
		}
	}

	if header != nil {
		res.Exists = true
		res.Size = header.Size
		res.Filename = header.Filename

		res.Content = make([]byte, res.Size)
		img.Read(res.Content)
		img.Seek(0, io.SeekStart)

		fileExtensionOverrides := []string{".svg"}
		fileExt := strings.ToLower(path.Ext(res.Filename))
		tryDecode := true
		for _, ext := range fileExtensionOverrides {
			if fileExt == ext {
				tryDecode = false
			}
		}

		if tryDecode {
			config, _, err := image.DecodeConfig(img)
			if err != nil {
				return FormImage{}, err
			}
			res.Width = config.Width
			res.Height = config.Height
			res.Mime = http.DetectContentType(res.Content)
		} else {
			if fileExt == ".svg" {
				res.Mime = "image/svg+xml"
			}
		}
	}

	return res, nil
}

func CanEditProject(user *models.User, owners []*models.User) bool {
	if user != nil {
		if user.IsStaff {
			return true
		} else {
			for _, owner := range owners {
				if owner.ID == user.ID {
					return true
				}
			}
		}
	}
	return false
}

package website

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

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

const maxPersonalProjects = 20
const maxProjectOwners = 5
const maxProjectScreenshots = 15

type ProjectTemplateData struct {
	templates.BaseData

	OfficialProjects []templates.Project
}

func ProjectIndex(c *RequestContext) ResponseData {
	officialProjects, err := getShuffledOfficialProjects(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	baseData := getBaseTemplateData(c, "Projects", nil)
	tmpl := ProjectTemplateData{
		BaseData: baseData,

		OfficialProjects: officialProjects,
	}

	var res ResponseData
	res.MustWriteTemplate("project_index.html", tmpl, c.Perf)
	return res
}

func getShuffledOfficialProjects(c *RequestContext) ([]templates.Project, error) {
	official, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		Types: hmndata.OfficialProjects,
	})
	if err != nil {
		return nil, oops.New(err, "failed to fetch projects")
	}

	defer c.Perf.StartBlock("PROJECT", "Grouping and sorting").End()

	var featuredProjects []hmndata.ProjectAndStuff
	var restProjects []hmndata.ProjectAndStuff
	for _, p := range official {
		if p.Project.Featured {
			featuredProjects = append(featuredProjects, p)
		} else {
			restProjects = append(restProjects, p)
		}
	}

	sort.Slice(featuredProjects, func(i, j int) bool {
		return featuredProjects[i].Project.SortScore > featuredProjects[j].Project.SortScore
	})
	sort.Slice(restProjects, func(i, j int) bool {
		return restProjects[i].Project.AllLastUpdated.After(restProjects[j].Project.AllLastUpdated)
	})

	projects := make([]templates.Project, 0, len(featuredProjects)+len(restProjects))
	for _, p := range featuredProjects {
		projects = append(projects, templates.ProjectAndStuffToTemplate(&p))
	}
	for _, p := range restProjects {
		projects = append(projects, templates.ProjectAndStuffToTemplate(&p))
	}

	return projects, nil
}

type ProjectPageBaseData struct {
	Owners   []templates.User
	Links    []templates.Link
	NavLinks []ProjectPageNavLink

	FollowUrl string
	Following bool
}

type ProjectPageNavLink struct {
	Name   string
	Url    string
	Active bool
}

func ProjectHomepage(c *RequestContext) ResponseData {
	if c.CurrentProject == nil {
		return FourOhFour(c)
	}

	type ProjectHomepageData struct {
		templates.BaseData
		ProjectPageBaseData

		Screenshots []string

		CanEdit bool
		EditUrl string
	}
	var tmpl ProjectHomepageData

	tmpl.BaseData = getBaseTemplateData(c, c.CurrentProject.Name, nil)
	tmpl.BaseData.OpenGraphItems = append(tmpl.BaseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    c.CurrentProject.Blurb,
	})

	projectBaseData, err := getProjectPageBaseData(c, &tmpl.BaseData, "Home")
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}
	tmpl.ProjectPageBaseData = projectBaseData

	screenshotAssets, err := db.Query[models.Asset](c, c.Conn,
		`
		---- Fetching screenshots
		SELECT $columns{asset}
		FROM
			project_screenshot
			JOIN asset ON project_screenshot.asset_id = asset.id
		WHERE
			project_screenshot.project_id = $1
		`,
		c.CurrentProject.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch screenshots for project"))
	}
	tmpl.Screenshots = utils.Map(screenshotAssets, templates.AssetUrl)

	tmpl.CanEdit = c.CurrentUserCanEditCurrentProject()
	tmpl.EditUrl = c.UrlContext.BuildProjectEdit("")

	if c.CurrentProject.Hidden {
		tmpl.BaseData.AddImmediateNotice(
			"hidden",
			"NOTICE: This project is hidden. It is currently visible only to owners and site admins.",
		)
	}

	if c.CurrentProject.Lifecycle != models.ProjectLifecycleActive {
		switch c.CurrentProject.Lifecycle {
		case models.ProjectLifecycleUnapproved:
			tmpl.BaseData.AddImmediateNotice(
				"unapproved",
				fmt.Sprintf(
					"NOTICE: This project has not yet been submitted for approval. It is only visible to owners. Please <a href=\"%s\">submit it for approval</a> when the project content is ready for review.",
					c.UrlContext.BuildProjectEdit("submit"),
				),
			)
		case models.ProjectLifecycleApprovalRequired:
			tmpl.BaseData.AddImmediateNotice(
				"unapproved",
				"NOTICE: This project is awaiting approval. It is only visible to owners and site admins.",
			)
		case models.ProjectLifecycleHiatus:
			tmpl.BaseData.AddImmediateNotice(
				"hiatus",
				"NOTICE: This project is on hiatus and may not update for a while.",
			)
		case models.ProjectLifecycleDead:
			tmpl.BaseData.AddImmediateNotice(
				"dead",
				"NOTICE: This project is has been marked dead and is only visible to owners and site admins.",
			)
		case models.ProjectLifecycleLTSRequired:
			tmpl.BaseData.AddImmediateNotice(
				"lts-reqd",
				"NOTICE: This project is awaiting approval for maintenance-mode status.",
			)
		}
	}

	// NOTE(ben): Prepare breadcrumb actions
	if c.CurrentUserCanEditCurrentProject() {
		tmpl.Header.Actions = append(tmpl.Header.Actions, templates.BreadcrumbAction{
			Name: "Edit Project",
			Url:  c.UrlContext.BuildProjectEdit(""),
			Icon: "edit-line",
		})
	}

	var res ResponseData
	err = res.WriteTemplate("project_homepage.html", tmpl, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render project homepage template"))
	}
	return res
}

func ProjectFeed(c *RequestContext) ResponseData {
	maxRecentActivity := 100

	type ProjectFeedData struct {
		templates.BaseData
		ProjectPageBaseData

		RecentActivity      []templates.TimelineItem
		SnippetEditorConfig templates.SnippetEditorConfig
	}
	var tmpl ProjectFeedData
	var err error

	tmpl.BaseData = getBaseTemplateData(c, "Feed", []templates.BreadcrumbLink{
		{Name: "Feed", Url: c.UrlContext.BuildProjectFeed()},
	})
	projectBaseData, err := getProjectPageBaseData(c, &tmpl.BaseData, "Feed")
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}
	tmpl.ProjectPageBaseData = projectBaseData

	// NOTE(ben): Get timeline activity
	{
		subforumTree := hmndata.GetFullSubforumTree(c, c.Conn)
		lineageBuilder := hmndata.MakeSubforumLineageBuilder(subforumTree)
		tmpl.RecentActivity, err = FetchTimeline(c, c.Conn, c.CurrentUser, lineageBuilder, hmndata.TimelineQuery{
			ProjectIDs: []int{c.CurrentProject.ID},
			Limit:      maxRecentActivity,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, err)
		}
	}

	// NOTE(ben): Prepare snippet (post) editor
	if c.CurrentUser != nil {
		userProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user projects"))
		}
		templateProjects := make([]templates.Project, 0, len(userProjects))
		templateProjects = append(templateProjects, templates.ProjectAndStuffToTemplate(&c.CurrentProjectExtras))
		for _, p := range userProjects {
			if p.Project.ID == c.CurrentProject.ID {
				continue
			}
			templateProject := templates.ProjectAndStuffToTemplate(&p)
			templateProjects = append(templateProjects, templateProject)
		}
		tmpl.SnippetEditorConfig = templates.SnippetEditorConfig{
			AssetMaxSize:      AssetMaxSize(c.CurrentUser),
			AvailableProjects: utils.Map(templateProjects, templates.ProjectToSnippetEditProject),
			Owner:             tmpl.User,
			RequiredProjectID: c.CurrentProject.ID,

			SubmitUrl: hmnurl.BuildSnippetSubmit(),
		}
	}

	var res ResponseData
	err = res.WriteTemplate("project_feed.html", tmpl, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render project homepage template"))
	}
	return res
}

func getProjectPageBaseData(c *RequestContext, base *templates.BaseData, activeLinkName string) (ProjectPageBaseData, error) {
	var res ProjectPageBaseData

	// NOTE(ben): Get project owners
	owners, err := hmndata.FetchProjectOwners(c, c.Conn, c.CurrentProject.ID)
	if err != nil {
		return ProjectPageBaseData{}, err
	}
	res.Owners = utils.Map(owners, templates.UserToTemplate)

	// NOTE(ben): Get user-created links
	{
		projectLinks, err := db.Query[models.Link](c, c.Conn,
			`
			---- Fetching project links
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
			return ProjectPageBaseData{}, oops.New(err, "failed to fetch project links")
		}
		res.Links = utils.Map(projectLinks, templates.LinkToTemplate)
	}

	// NOTE(ben): Get nav links
	{
		res.NavLinks = append(res.NavLinks, ProjectPageNavLink{
			Name: "Home",
			Url:  c.UrlContext.BuildHomepage(),
		})
		res.NavLinks = append(res.NavLinks, ProjectPageNavLink{
			Name: "Feed",
			Url:  c.UrlContext.BuildProjectFeed(),
		})
		if c.CurrentProject.HasBlog() {
			canSeeBlogLink := false
			if c.CurrentUser != nil {
				if c.CurrentUser.IsStaff {
					canSeeBlogLink = true
				} else {
					for _, owner := range owners {
						if owner.ID == c.CurrentUser.ID {
							canSeeBlogLink = true
							break
						}
					}
				}
			}

			if !canSeeBlogLink {
				hasBlogPosts, err := db.QueryOneScalar[bool](c, c.Conn,
					`
					---- Check for blog posts
					SELECT COUNT(*) > 0
					FROM thread
					WHERE
						type = $1
						AND project_id = $2
						AND deleted = false
					`,
					models.ThreadTypeProjectBlogPost,
					c.CurrentProject.ID,
				)
				if err != nil {
					return ProjectPageBaseData{}, oops.New(err, "failed to fetch project blogs")
				}

				canSeeBlogLink = hasBlogPosts
			}

			if canSeeBlogLink {
				res.NavLinks = append(res.NavLinks, ProjectPageNavLink{
					Name: "Blog",
					Url:  c.UrlContext.BuildBlog(1),
				})
			}
		}

		for i := range res.NavLinks {
			if res.NavLinks[i].Name == activeLinkName {
				res.NavLinks[i].Active = true
			}
		}
	}

	// NOTE(ben): Get header actions (follow/unfollow)
	if c.CurrentUser != nil {
		res.FollowUrl = hmnurl.BuildFollowProject()
		res.Following, err = db.QueryOneScalar[bool](c, c.Conn, `
			---- Check following
			SELECT COUNT(*) > 0
			FROM follower
			WHERE user_id = $1 AND following_project_id = $2
		`, c.CurrentUser.ID, c.CurrentProject.ID)
		if err != nil {
			return ProjectPageBaseData{}, oops.New(err, "failed to fetch following status")
		}

		if res.Following {
			base.Header.Actions = append(base.Header.Actions, templates.BreadcrumbAction{
				Name: "Unfollow",
				Url:  res.FollowUrl,
				Icon: "remove",

				PostData: []templates.BreadcrumbActionPostData{
					{"project_id", c.CurrentProject.ID},
					{"redirect", c.FullUrl()},
					{"unfollow", true},
				},
			})
		} else {
			base.Header.Actions = append(base.Header.Actions, templates.BreadcrumbAction{
				Name: "Follow",
				Url:  res.FollowUrl,
				Icon: "add",

				PostData: []templates.BreadcrumbActionPostData{
					{"project_id", c.CurrentProject.ID},
					{"redirect", c.FullUrl()},
				},
			})
		}
	}

	return res, nil
}

var ProjectLogoMaxFileSize = 2 * 1024 * 1024

type ProjectEditData struct {
	templates.BaseData

	Editing         bool
	ProjectSettings templates.ProjectSettings
	MaxOwners       int
	MaxScreenshots  int

	APICheckUsernameUrl   string
	LogoMaxFileSize       int
	ScreenshotMaxFileSize int

	AllLogos []templates.Icon

	TextEditor templates.TextEditor

	DiscordSettingsUrl string
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

	currentJam := hmndata.UpcomingJam(hmndata.JamProjectCreateGracePeriod)
	if currentJam != nil {
		project.JamParticipation = []templates.ProjectJamParticipation{
			{
				JamName:       currentJam.Name,
				JamSlug:       currentJam.Slug,
				Participating: c.Req.URL.Query().Has("jam"),
			},
		}
	}

	var res ResponseData
	res.MustWriteTemplate("project_edit.html", ProjectEditData{
		BaseData:        getBaseTemplateData(c, "New Project", nil),
		Editing:         false,
		ProjectSettings: project,
		MaxOwners:       maxProjectOwners,
		MaxScreenshots:  maxProjectScreenshots,

		APICheckUsernameUrl:   hmnurl.BuildAPICheckUsername(),
		LogoMaxFileSize:       ProjectLogoMaxFileSize,
		ScreenshotMaxFileSize: AssetMaxSize(c.CurrentUser),

		AllLogos: allLogos(),

		TextEditor: templates.TextEditor{
			MaxFileSize: AssetMaxSize(c.CurrentUser),
			UploadUrl:   c.UrlContext.BuildAssetUpload(),
		},

		DiscordSettingsUrl: hmnurl.BuildUserSettings("discord"),
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
	if !c.CurrentUserCanEditCurrentProject() {
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

	projectLinks, err := db.Query[models.Link](c, c.Conn,
		`
		---- Fetching project links
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

	projectScreenshots, err := db.Query[models.Asset](c, c.Conn,
		`
		---- Fetching project screenshots
		SELECT $columns{asset}
		FROM
			project_screenshot
			JOIN asset ON project_screenshot.asset_id = asset.id
		WHERE
			project_screenshot.project_id = $1
		ORDER BY
			project_screenshot.sort
		`,
		p.Project.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project screenshots"))
	}

	projectJams, err := hmndata.FetchJamsForProject(c, c.Conn, c.CurrentUser, p.Project.ID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jams for project"))
	}

	projectSettings := templates.ProjectToProjectSettings(
		&p.Project,
		p.Owners,
		p.TagText(),
		projectJams,
		projectLinks,
		p.LogoAsset, p.HeaderImage,
		projectScreenshots,
	)

	var res ResponseData
	res.MustWriteTemplate("project_edit.html", ProjectEditData{
		BaseData:        getBaseTemplateData(c, "Edit Project", nil),
		Editing:         true,
		ProjectSettings: projectSettings,
		MaxOwners:       maxProjectOwners,
		MaxScreenshots:  maxProjectScreenshots,

		APICheckUsernameUrl:   hmnurl.BuildAPICheckUsername(),
		LogoMaxFileSize:       ProjectLogoMaxFileSize,
		ScreenshotMaxFileSize: AssetMaxSize(c.CurrentUser),

		AllLogos: allLogos(),

		TextEditor: templates.TextEditor{
			MaxFileSize: AssetMaxSize(c.CurrentUser),
			UploadUrl:   c.UrlContext.BuildAssetUpload(),
		},

		DiscordSettingsUrl: hmnurl.BuildUserSettings("discord"),
	}, c.Perf)
	return res
}

func ProjectEditSubmit(c *RequestContext) ResponseData {
	if !c.CurrentUserCanEditCurrentProject() {
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
	Logo                  FormImage
	Screenshots           []FormImage
	Tag                   string
	JamParticipationSlugs []string
	JamHidden             bool
	SortScore             int

	Slug        string
	SlugAliases string // comma-separated
	Featured    bool
	Personal    bool
}

type ProjectEditFormResult struct {
	Payload         ProjectPayload
	RejectionReason string
	Error           error
}

func ParseProjectEditForm(c *RequestContext) ProjectEditFormResult {
	var res ProjectEditFormResult
	maxBodySize := int64(ProjectLogoMaxFileSize + 1024*1024)
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
	parsedDescription := parsing.ParseMarkdown(description, parsing.PostMarkdown)

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

	logo, err := GetFormImage(c, "logo")
	if err != nil {
		res.Error = oops.New(err, "Failed to read image from form")
		return res
	}
	screenshots, err := GetFormImages(c, "screenshot")
	if err != nil {
		res.Error = oops.New(err, "Failed to read screenshots from form")
		return res
	}

	owners := c.Req.Form["owners"]
	if len(owners) > maxProjectOwners {
		res.RejectionReason = fmt.Sprintf("Projects can have at most %d owners", maxProjectOwners)
		return res
	}

	slug := strings.TrimSpace(c.Req.Form.Get("slug"))
	slugAliases := c.Req.Form.Get("slug_aliases")
	officialStr := c.Req.Form.Get("official")
	official := len(officialStr) > 0
	featuredStr := c.Req.Form.Get("featured")
	featured := len(featuredStr) > 0

	if official && len(slug) == 0 {
		res.RejectionReason = "Official projects must have a slug"
		return res
	}

	jamParticipationSlugs := c.Req.Form["jam_participation"]
	jamHidden := c.Req.Form.Has("jam_hidden")

	sortScoreStr := c.Req.Form.Get("sort_score")
	sortScore, _ := strconv.Atoi(sortScoreStr)

	res.Payload = ProjectPayload{
		Name:                  projectName,
		Blurb:                 shortDesc,
		Links:                 links,
		Description:           description,
		ParsedDescription:     parsedDescription,
		Lifecycle:             lifecycle,
		Hidden:                hidden,
		OwnerUsernames:        owners,
		Logo:                  logo,
		Screenshots:           screenshots,
		Tag:                   tag,
		JamParticipationSlugs: jamParticipationSlugs,
		JamHidden:             jamHidden,
		Slug:                  slug,
		SlugAliases:           slugAliases,
		Personal:              !official,
		Featured:              featured,
		SortScore:             sortScore,
	}

	return res
}

func updateProject(ctx context.Context, tx pgx.Tx, user *models.User, payload *ProjectPayload) error {
	numNewOrExistingScreenshots := 0
	for _, screenshot := range payload.Screenshots {
		if screenshot.New || screenshot.Exists {
			numNewOrExistingScreenshots += 1
		}
	}
	if numNewOrExistingScreenshots > maxProjectScreenshots {
		return errors.New("too many screenshots")
	}

	// NOTE(ben): Upload all new assets before proceeding with DB updates.
	var logoUUID *uuid.UUID
	var newScreenshotUUIDs []uuid.UUID
	if payload.Logo.New {
		logoAsset, err := SaveFormImage(ctx, tx, payload.Logo, &user.ID)
		if err != nil {
			return oops.New(err, "Failed to save asset")
		}
		logoUUID = &logoAsset.ID
	}
	for _, screenshot := range payload.Screenshots {
		if screenshot.New {
			screenshotAsset, err := SaveFormImage(ctx, tx, screenshot, &user.ID)
			if err != nil {
				return oops.New(err, "Failed to save screenshot")
			}
			newScreenshotUUIDs = append(newScreenshotUUIDs, screenshotAsset.ID)
		}
	}

	hasSelf := false
	selfUsername := strings.ToLower(user.Username)
	for i := range payload.OwnerUsernames {
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
		slugAliases := strings.Split(payload.SlugAliases, ",")
		for i := range slugAliases {
			slugAliases[i] = strings.TrimSpace(slugAliases[i])
		}

		_, err = tx.Exec(ctx,
			`
			UPDATE project SET
				slug = $2,
				featured = $3,
				personal = $4,
				hidden = $5,
				slug_aliases = $6,
				jam_hidden = $7,
				sort_score = $8
			WHERE
				id = $1
			`,
			payload.ProjectID,
			payload.Slug,
			payload.Featured,
			payload.Personal,
			payload.Hidden,
			slugAliases,
			payload.JamHidden,
			payload.SortScore,
		)
		if err != nil {
			return oops.New(err, "Failed to update project with admin fields")
		}
	}

	// NOTE(ben): Update images and screenshots
	{
		var errs []error

		if payload.Logo.New || payload.Logo.Remove {
			_, err = tx.Exec(ctx,
				`
				UPDATE project
				SET
					logo_asset_id = $2
				WHERE
					id = $1
				`,
				payload.ProjectID,
				logoUUID,
			)
			if err != nil {
				errs = append(errs, oops.New(err, "Failed to update project's logo"))
			}
		}

		currentNewScreenshot := 0
		for sort, screenshot := range payload.Screenshots {
			if screenshot.New || screenshot.Exists {
				assetID := screenshot.AssetID
				if screenshot.New {
					// NOTE(ben): No bounds check necessary because newScreenshotUUIDs was
					// generated from screenshot.New's above.
					assetID = newScreenshotUUIDs[currentNewScreenshot].String()
					currentNewScreenshot++
				}

				_, err := tx.Exec(ctx,
					`
					---- Upsert project screenshot
					INSERT INTO project_screenshot (project_id, asset_id, sort)
					VALUES ($1, $2, $3)
					ON CONFLICT (project_id, asset_id) DO UPDATE
						SET sort = $3
					`,
					payload.ProjectID, assetID, sort,
				)
				if err != nil {
					errs = append(errs, err)
					continue
				}
			} else {
				utils.Assert(screenshot.Remove)
				utils.Assert(screenshot.AssetID)

				res, err := tx.Exec(ctx,
					`
					---- Delete project screenshot
					DELETE FROM project_screenshot
					WHERE project_id = $1 AND asset_id = $2
					`,
					payload.ProjectID, screenshot.AssetID,
				)
				if err != nil {
					errs = append(errs, err)
					continue
				}
				if res.RowsAffected() != 1 {
					errs = append(errs, errors.New("attempt to delete nonexistent screenshot"))
					continue
				}
			}
		}

		if err := errors.Join(errs...); err != nil {
			return oops.New(err, "failed to upload project images")
		}

		// NOTE(ben): Sanity check: After all this, we should have N project
		// screenshots that all have different sorts.
		type Sanity struct {
			Total    int `db:"COUNT(*)"`
			NumSorts int `db:"COUNT(DISTINCT sort)"`
		}
		sanity := utils.Must1(db.QueryOne[Sanity](ctx, tx,
			`
			---- Project screenshot sanity check
			SELECT $columns FROM project_screenshot
			WHERE project_id = $1
			`,
			payload.ProjectID,
		))
		utils.Assert(
			sanity.Total == numNewOrExistingScreenshots && sanity.NumSorts == sanity.Total,
			fmt.Sprintf("should have %d screenshots, but had %d with %d distinct sorts", len(payload.Screenshots), sanity.Total, sanity.NumSorts),
		)
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
			INSERT INTO link (name, url, ordering, primary_link, project_id)
			VALUES ($1, $2, $3, $4, $5)
			`,
			link.Name,
			link.Url,
			i,
			link.Primary,
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
		currentJam := hmndata.UpcomingJam(hmndata.JamProjectCreateGracePeriod)
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
		found := slices.Contains(possibleJamSlugs, jamSlug)
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

func allLogos() []templates.Icon {
	var logos []templates.Icon
	logoEntries := templates.ListImgsDir("logos")
	for _, logo := range logoEntries {
		logos = append(logos, templates.Icon{
			Name: logo.Name()[:len(logo.Name())-len(".svg")],
			Svg:  template.HTML(templates.GetImg(fmt.Sprintf("logos/%s", logo.Name()))),
		})
	}

	return logos
}

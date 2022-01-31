package website

import (
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

func getBaseDataAutocrumb(c *RequestContext, title string) templates.BaseData {
	return getBaseData(c, title, []templates.Breadcrumb{{Name: title, Url: ""}})
}

// NOTE(asaf): If you set breadcrumbs, the breadcrumb for the current project will automatically be prepended when necessary.
//             If you pass nil, no breadcrumbs will be created.
func getBaseData(c *RequestContext, title string, breadcrumbs []templates.Breadcrumb) templates.BaseData {
	var project models.Project
	if c.CurrentProject != nil {
		project = *c.CurrentProject
	}

	var templateUser *templates.User
	var templateSession *templates.Session
	if c.CurrentUser != nil {
		u := templates.UserToTemplate(c.CurrentUser, c.Theme)
		s := templates.SessionToTemplate(c.CurrentSession)
		templateUser = &u
		templateSession = &s
	}

	notices := getNoticesFromCookie(c)

	if len(breadcrumbs) > 0 {
		projectUrl := c.UrlContext.BuildHomepage()
		if breadcrumbs[0].Url != projectUrl {
			rootBreadcrumb := templates.Breadcrumb{
				Name: project.Name,
				Url:  projectUrl,
			}
			breadcrumbs = append([]templates.Breadcrumb{rootBreadcrumb}, breadcrumbs...)
		}
	}

	baseData := templates.BaseData{
		Theme:       c.Theme,
		Title:       title,
		Breadcrumbs: breadcrumbs,

		CurrentUrl:        c.FullUrl(),
		CurrentProjectUrl: c.UrlContext.BuildHomepage(),
		LoginPageUrl:      hmnurl.BuildLoginPage(c.FullUrl()),
		ProjectCSSUrl:     hmnurl.BuildProjectCSS(project.Color1),

		Project: templates.ProjectToTemplate(&project, c.UrlContext.BuildHomepage()),
		User:    templateUser,
		Session: templateSession,
		Notices: notices,

		ReportIssueMailto: "team@handmade.network",

		OpenGraphItems: buildDefaultOpenGraphItems(&project, title),

		IsProjectPage: !project.IsHMN(),
		Header: templates.Header{
			AdminUrl:          hmnurl.BuildAdminApprovalQueue(), // TODO(asaf): Replace with general-purpose admin page
			UserSettingsUrl:   hmnurl.BuildUserSettings(""),
			LoginActionUrl:    hmnurl.BuildLoginAction(c.FullUrl()),
			LogoutActionUrl:   hmnurl.BuildLogoutAction(c.FullUrl()),
			ForgotPasswordUrl: hmnurl.BuildRequestPasswordReset(),
			RegisterUrl:       hmnurl.BuildRegister(),

			HMNHomepageUrl:  hmnurl.BuildHomepage(),
			ProjectIndexUrl: hmnurl.BuildProjectIndex(1),
			PodcastUrl:      hmnurl.BuildPodcast(),
			ForumsUrl:       hmnurl.HMNProjectContext.BuildForum(nil, 1),
			LibraryUrl:      hmnurl.BuildLibrary(),
		},
		Footer: templates.Footer{
			HomepageUrl:                hmnurl.BuildHomepage(),
			AboutUrl:                   hmnurl.BuildAbout(),
			ManifestoUrl:               hmnurl.BuildManifesto(),
			CodeOfConductUrl:           hmnurl.BuildCodeOfConduct(),
			CommunicationGuidelinesUrl: hmnurl.BuildCommunicationGuidelines(),
			ProjectIndexUrl:            hmnurl.BuildProjectIndex(1),
			ForumsUrl:                  hmnurl.HMNProjectContext.BuildForum(nil, 1),
			ContactUrl:                 hmnurl.BuildContactPage(),
		},
	}

	if c.CurrentUser != nil {
		baseData.Header.UserProfileUrl = hmnurl.BuildUserProfile(c.CurrentUser.Username)
	}

	if !project.IsHMN() {
		episodeGuideUrl := ""
		defaultTopic, hasAnnotations := config.Config.EpisodeGuide.Projects[project.Slug]
		if hasAnnotations {
			episodeGuideUrl = c.UrlContext.BuildEpisodeList(defaultTopic)
		}

		baseData.Header.Project = &templates.ProjectHeader{
			HasForums:       project.HasForums(),
			HasBlog:         project.HasBlog(),
			HasEpisodeGuide: hasAnnotations,
			CanEdit:         c.CurrentUserCanEditCurrentProject,
			ForumsUrl:       c.UrlContext.BuildForum(nil, 1),
			BlogUrl:         c.UrlContext.BuildBlog(1),
			EpisodeGuideUrl: episodeGuideUrl,
			EditUrl:         c.UrlContext.BuildProjectEdit(""),
		}
	}

	return baseData
}

func buildDefaultOpenGraphItems(project *models.Project, title string) []templates.OpenGraphItem {
	if title == "" {
		title = "Handmade Network"
	}

	image := hmnurl.BuildPublic("logo.png", false)
	if !project.IsHMN() {
		image = hmnurl.BuildUserFile(project.LogoLight)
	}

	return []templates.OpenGraphItem{
		{Property: "og:title", Value: title},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: image},
	}
}
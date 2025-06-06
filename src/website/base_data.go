package website

import (
	"time"

	"git.handmade.network/hmn/hmn/src/buildcss"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

// NOTE(asaf): If you set breadcrumbs, the breadcrumb for the current project will automatically be prepended when necessary.
//
//	If you pass nil, no breadcrumbs will be created.
func getBaseData(c *RequestContext, title string, breadcrumbs []templates.Breadcrumb) templates.BaseData {
	var project models.Project
	if c.CurrentProject != nil {
		project = *c.CurrentProject
	}

	var templateUser *templates.User
	var templateSession *templates.Session
	if c.CurrentUser != nil {
		u := templates.UserToTemplate(c.CurrentUser)
		s := templates.SessionToTemplate(c.CurrentSession)
		templateUser = &u
		templateSession = &s
	}
	templateProject := templates.ProjectToTemplate(&project)
	templateProject.Logo = c.CurrentProjectLogoUrl

	notices := getNoticesFromCookie(c)

	if !c.UrlContext.IsHMN() {
		projectUrl := c.UrlContext.BuildHomepage()
		rootBreadcrumb := templates.Breadcrumb{
			Name:    project.Name,
			Url:     projectUrl,
			Project: &templateProject,
		}
		breadcrumbs = append([]templates.Breadcrumb{rootBreadcrumb}, breadcrumbs...)

		if len(breadcrumbs) > 1 && breadcrumbs[1].Url == projectUrl {
			c.Logger.Warn().Msg("duplicate root breadcrumb")
		}
	}

	var bannerEvent *templates.BannerEvent
	for _, jam := range hmndata.AllJams {
		if jam.Event.WithinGrace(time.Now(), JamBannerGraceBefore, JamBannerGraceAfter) {
			bannerEvent = utils.P(templates.JamToBannerEvent(jam))
		}
	}

	baseData := templates.BaseData{
		Title: title,

		CurrentUrl:          c.FullUrl(),
		CurrentProjectUrl:   c.UrlContext.BuildHomepage(),
		LoginPageUrl:        hmnurl.BuildLoginPage(c.FullUrl()),
		DiscordInviteUrl:    "https://discord.gg/hmn",
		NewsletterSignupUrl: hmnurl.BuildAPINewsletterSignup(),

		Project: templateProject,
		User:    templateUser,
		Session: templateSession,
		Notices: notices,

		ReportIssueEmail: "team@handmade.network",

		OpenGraphItems: buildDefaultOpenGraphItems(&project, c.CurrentProjectLogoUrl, title),

		IsProjectPage: !project.IsHMN(),
		Header: templates.Header{
			AdminApprovalQueueUrl: hmnurl.BuildAdminApprovalQueue(), // TODO(asaf): Replace with general-purpose admin page
			UserSettingsUrl:       hmnurl.BuildUserSettings(""),
			LogoutUrl:             hmnurl.BuildLogoutAction(c.FullUrl()),
			ForgotPasswordUrl:     hmnurl.BuildRequestPasswordReset(),
			RegisterUrl:           hmnurl.BuildRegister(""),

			HMNHomepageUrl:  hmnurl.BuildHomepage(),
			ProjectIndexUrl: hmnurl.BuildProjectIndex(),
			PodcastUrl:      hmnurl.BuildPodcast(),
			NewsletterUrl:   hmnurl.BuildNewsletterSignup(),
			FishbowlUrl:     hmnurl.BuildFishbowlIndex(),
			ForumsUrl:       hmnurl.HMNProjectContext.BuildForum(nil, 1),
			JamsUrl:         hmnurl.BuildJamsIndex(),
			EducationUrl:    hmnurl.BuildEducationIndex(),
			CalendarUrl:     hmnurl.BuildCalendarIndex(),
			ManifestoUrl:    hmnurl.BuildManifesto(),
			AboutUrl:        hmnurl.BuildAbout(),

			Breadcrumbs: breadcrumbs,

			BannerEvent: bannerEvent,
		},
		Footer: templates.Footer{
			HomepageUrl:                hmnurl.BuildHomepage(),
			AboutUrl:                   hmnurl.BuildAbout(),
			ManifestoUrl:               hmnurl.BuildManifesto(),
			CommunicationGuidelinesUrl: hmnurl.BuildCommunicationGuidelines(),
			ProjectIndexUrl:            hmnurl.BuildProjectIndex(),
			ContactUrl:                 hmnurl.BuildContactPage(),
			SearchActionUrl:            "https://duckduckgo.com",
		},
	}

	if buildcss.ActiveServerPort != 0 {
		baseData.EsBuildSSEUrl = hmnurl.BuildEsBuild()
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

func buildDefaultOpenGraphItems(project *models.Project, projectLogoUrl string, title string) []templates.OpenGraphItem {
	if title == "" {
		title = "Handmade Network"
	}

	image := hmnurl.BuildPublic("logo.png", false)
	if !project.IsHMN() {
		image = projectLogoUrl
	}

	return []templates.OpenGraphItem{
		{Property: "og:title", Value: title},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: image},
	}
}

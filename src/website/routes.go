package website

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NewWebsiteRoutes(conn *pgxpool.Pool) http.Handler {
	router := &Router{}
	routes := RouteBuilder{
		Router: router,
		Middlewares: []Middleware{
			setDBConn(conn),
			trackRequestPerf,
			logContextErrorsMiddleware,
			panicCatcherMiddleware,
		},
	}

	anyProject := routes.WithMiddleware(
		storeNoticesInCookieMiddleware,
		loadCommonData,
	)
	hmnOnly := anyProject.WithMiddleware(
		redirectToHMN,
	)

	routes.GET(hmnurl.RegexPublic, func(c *RequestContext) ResponseData {
		var res ResponseData
		http.StripPrefix("/public/", http.FileServer(http.Dir("public"))).ServeHTTP(&res, c.Req)
		addCORSHeaders(c, &res)
		return res
	})
	routes.GET(hmnurl.RegexFishbowlFiles, FishbowlFiles)

	// NOTE(asaf): HMN-only routes:
	hmnOnly.GET(hmnurl.RegexManifesto, Manifesto)
	hmnOnly.GET(hmnurl.RegexAbout, About)
	// hmnOnly.GET(hmnurl.RegexFoundation, Foundation)
	hmnOnly.GET(hmnurl.RegexCommunicationGuidelines, CommunicationGuidelines)
	hmnOnly.GET(hmnurl.RegexContactPage, ContactPage)
	hmnOnly.GET(hmnurl.RegexMonthlyUpdatePolicy, MonthlyUpdatePolicy)
	hmnOnly.GET(hmnurl.RegexProjectSubmissionGuidelines, ProjectSubmissionGuidelines)
	hmnOnly.GET(hmnurl.RegexConferences, Conferences)
	hmnOnly.GET(hmnurl.RegexWhenIsIt, WhenIsIt)

	hmnOnly.GET(hmnurl.RegexJamsIndex, JamsIndex)
	hmnOnly.GET(hmnurl.RegexJamIndex, func(c *RequestContext) ResponseData {
		return c.Redirect(hmnurl.BuildJamSaveTheDate(), http.StatusFound)
	})
	hmnOnly.GET(hmnurl.RegexJamIndex2021, JamIndex2021)
	hmnOnly.GET(hmnurl.RegexJamIndex2022, JamIndex2022)
	hmnOnly.GET(hmnurl.RegexJamFeed2022, JamFeed2022)
	hmnOnly.GET(hmnurl.RegexJamIndex2023_Visibility, JamIndex2023_Visibility)
	hmnOnly.GET(hmnurl.RegexJamFeed2023_Visibility, JamFeed2023_Visibility)
	hmnOnly.GET(hmnurl.RegexJamRecap2023_Visibility, JamRecap2023_Visibility)
	hmnOnly.GET(hmnurl.RegexJamIndex2023, JamIndex2023)
	hmnOnly.GET(hmnurl.RegexJamFeed2023, JamFeed2023)
	hmnOnly.GET(hmnurl.RegexJamIndex2024_Learning, JamIndex2024_Learning)
	hmnOnly.GET(hmnurl.RegexJamFeed2024_Learning, JamFeed2024_Learning)
	hmnOnly.GET(hmnurl.RegexJamGuidelines2024_Learning, JamGuidelines2024_Learning)
	hmnOnly.GET(hmnurl.RegexJamSaveTheDate, JamSaveTheDate)

	hmnOnly.GET(hmnurl.RegexTimeMachine, TimeMachine)
	hmnOnly.GET(hmnurl.RegexTimeMachineSubmissions, TimeMachineSubmissions)
	hmnOnly.GET(hmnurl.RegexTimeMachineAtomFeed, TimeMachineAtomFeed)
	hmnOnly.GET(hmnurl.RegexTimeMachineForm, needsAuth(TimeMachineForm))
	hmnOnly.GET(hmnurl.RegexTimeMachineFormDone, needsAuth(TimeMachineFormDone))
	hmnOnly.POST(hmnurl.RegexTimeMachineForm, needsAuth(csrfMiddleware(TimeMachineFormSubmit)))

	hmnOnly.GET(hmnurl.RegexCalendarIndex, CalendarIndex)
	hmnOnly.GET(hmnurl.RegexCalendarICal, CalendarICal)

	hmnOnly.GET(hmnurl.RegexStaffRolesIndex, StaffRolesIndex)
	hmnOnly.GET(hmnurl.RegexStaffRole, StaffRole)

	hmnOnly.GET(hmnurl.RegexOldHome, Index)

	hmnOnly.POST(hmnurl.RegexLoginAction, securityTimerMiddleware(time.Millisecond*100, Login))
	hmnOnly.GET(hmnurl.RegexLogoutAction, Logout)
	hmnOnly.GET(hmnurl.RegexLoginPage, LoginPage)
	hmnOnly.GET(hmnurl.RegexLoginWithDiscord, LoginWithDiscord)

	hmnOnly.GET(hmnurl.RegexRegister, RegisterNewUser)
	hmnOnly.POST(hmnurl.RegexRegister, securityTimerMiddleware(email.ExpectedEmailSendDuration, RegisterNewUserSubmit))
	hmnOnly.GET(hmnurl.RegexRegistrationSuccess, RegisterNewUserSuccess)
	hmnOnly.GET(hmnurl.RegexEmailConfirmation, EmailConfirmation)
	hmnOnly.POST(hmnurl.RegexEmailConfirmation, EmailConfirmationSubmit)

	hmnOnly.GET(hmnurl.RegexRequestPasswordReset, RequestPasswordReset)
	hmnOnly.POST(hmnurl.RegexRequestPasswordReset, securityTimerMiddleware(email.ExpectedEmailSendDuration, RequestPasswordResetSubmit))
	hmnOnly.GET(hmnurl.RegexPasswordResetSent, PasswordResetSent)
	hmnOnly.GET(hmnurl.RegexOldDoPasswordReset, DoPasswordReset)
	hmnOnly.GET(hmnurl.RegexDoPasswordReset, DoPasswordReset)
	hmnOnly.POST(hmnurl.RegexDoPasswordReset, DoPasswordResetSubmit)

	hmnOnly.GET(hmnurl.RegexAdminAtomFeed, AdminAtomFeed)
	hmnOnly.GET(hmnurl.RegexAdminApprovalQueue, adminsOnly(AdminApprovalQueue))
	hmnOnly.POST(hmnurl.RegexAdminApprovalQueue, adminsOnly(csrfMiddleware(AdminApprovalQueueSubmit)))
	hmnOnly.POST(hmnurl.RegexAdminSetUserOptions, adminsOnly(csrfMiddleware(UserProfileAdminSetOptions)))
	hmnOnly.POST(hmnurl.RegexAdminNukeUser, adminsOnly(csrfMiddleware(UserProfileAdminNuke)))

	hmnOnly.GET(hmnurl.RegexFeed, Feed)
	hmnOnly.GET(hmnurl.RegexAtomFeed, AtomFeed)
	hmnOnly.GET(hmnurl.RegexShowcase, Showcase)
	hmnOnly.GET(hmnurl.RegexSnippet, Snippet)
	hmnOnly.GET(hmnurl.RegexProjectIndex, ProjectIndex)

	hmnOnly.GET(hmnurl.RegexProjectNew, needsAuth(ProjectNew))
	hmnOnly.POST(hmnurl.RegexProjectNew, needsAuth(csrfMiddleware(ProjectNewSubmit)))

	hmnOnly.GET(hmnurl.RegexDiscordOAuthCallback, DiscordOAuthCallback)
	hmnOnly.POST(hmnurl.RegexDiscordUnlink, needsAuth(csrfMiddleware(DiscordUnlink)))
	hmnOnly.POST(hmnurl.RegexDiscordShowcaseBacklog, needsAuth(csrfMiddleware(DiscordShowcaseBacklog)))
	hmnOnly.GET(hmnurl.RegexDiscordBotDebugPage, adminsOnly(DiscordBotDebugPage))

	hmnOnly.POST(hmnurl.RegexTwitchEventSubCallback, TwitchEventSubCallback)
	hmnOnly.GET(hmnurl.RegexTwitchDebugPage, adminsOnly(TwitchDebugPage))

	hmnOnly.GET(hmnurl.RegexUserProfile, UserProfile)
	hmnOnly.GET(hmnurl.RegexUserSettings, needsAuth(UserSettings))
	hmnOnly.POST(hmnurl.RegexUserSettings, needsAuth(csrfMiddleware(UserSettingsSave)))

	hmnOnly.GET(hmnurl.RegexPodcast, PodcastIndex)
	hmnOnly.GET(hmnurl.RegexPodcastEdit, PodcastEdit)
	hmnOnly.POST(hmnurl.RegexPodcastEdit, PodcastEditSubmit)
	hmnOnly.GET(hmnurl.RegexPodcastEpisodeNew, PodcastEpisodeNew)
	hmnOnly.POST(hmnurl.RegexPodcastEpisodeNew, PodcastEpisodeSubmit)
	hmnOnly.GET(hmnurl.RegexPodcastEpisodeEdit, PodcastEpisodeEdit)
	hmnOnly.POST(hmnurl.RegexPodcastEpisodeEdit, PodcastEpisodeSubmit)
	hmnOnly.GET(hmnurl.RegexPodcastEpisode, PodcastEpisode)
	hmnOnly.GET(hmnurl.RegexPodcastRSS, PodcastRSS)

	hmnOnly.GET(hmnurl.RegexFishbowlIndex, FishbowlIndex)
	hmnOnly.GET(hmnurl.RegexFishbowl, Fishbowl)

	hmnOnly.GET(hmnurl.RegexEducationIndex, EducationIndex)
	hmnOnly.GET(hmnurl.RegexEducationGlossary, educationAuthorsOnly(EducationGlossary)) // TODO: Remove this gate
	hmnOnly.GET(hmnurl.RegexEducationArticleNew, educationAuthorsOnly(EducationArticleNew))
	hmnOnly.POST(hmnurl.RegexEducationArticleNew, educationAuthorsOnly(EducationArticleNewSubmit))
	hmnOnly.GET(hmnurl.RegexEducationRerender, educationAuthorsOnly(EducationRerender))
	hmnOnly.GET(hmnurl.RegexEducationArticle, EducationArticle) // Article stuff must be last so `/glossary` and others do not match as an article slug
	hmnOnly.GET(hmnurl.RegexEducationArticleEdit, educationAuthorsOnly(EducationArticleEdit))
	hmnOnly.POST(hmnurl.RegexEducationArticleEdit, educationAuthorsOnly(EducationArticleEditSubmit))
	hmnOnly.GET(hmnurl.RegexEducationArticleDelete, educationAuthorsOnly(EducationArticleDelete))
	hmnOnly.POST(hmnurl.RegexEducationArticleDelete, educationAuthorsOnly(csrfMiddleware(EducationArticleDeleteSubmit)))

	hmnOnly.POST(hmnurl.RegexAPICheckUsername, csrfMiddleware(APICheckUsername))
	hmnOnly.POST(hmnurl.RegexAPINewsletterSignup, APINewsletterSignup)

	hmnOnly.GET(hmnurl.RegexLibraryAny, func(c *RequestContext) ResponseData {
		return c.Redirect(hmnurl.BuildEducationIndex(), http.StatusFound)
	})

	hmnOnly.GET(hmnurl.RegexUnwind, func(c *RequestContext) ResponseData {
		return c.Redirect("https://www.youtube.com/playlist?list=PL-IPpPzBYXBGsAd9-c2__x6LJG4Zszs0T", http.StatusFound)
	})

	// Project routes can appear either at the root (e.g. hero.handmade.network/edit)
	// or on a personal project path (e.g. handmade.network/p/123/hero/edit). So, we
	// have pulled all those routes into this function.
	attachProjectRoutes := func(rb *RouteBuilder) {
		rb.GET(hmnurl.RegexHomepage, func(c *RequestContext) ResponseData {
			if c.CurrentProject.IsHMN() {
				return Index(c)
			} else {
				return ProjectHomepage(c)
			}
		})

		rb.GET(hmnurl.RegexProjectEdit, needsAuth(ProjectEdit))
		rb.POST(hmnurl.RegexProjectEdit, needsAuth(csrfMiddleware(ProjectEditSubmit)))

		// Middleware used for forum action routes - anything related to actually creating or editing forum content
		needsForums := func(h Handler) Handler {
			return func(c *RequestContext) ResponseData {
				// 404 if the project has forums disabled
				if !c.CurrentProject.HasForums() {
					return FourOhFour(c)
				}
				// Require auth if forums are enabled
				return needsAuth(h)(c)
			}
		}
		rb.POST(hmnurl.RegexForumNewThreadSubmit, needsForums(csrfMiddleware(ForumNewThreadSubmit)))
		rb.GET(hmnurl.RegexForumNewThread, needsForums(ForumNewThread))
		rb.GET(hmnurl.RegexForumThread, ForumThread)
		rb.GET(hmnurl.RegexForum, Forum)
		rb.POST(hmnurl.RegexForumMarkRead, needsAuth(csrfMiddleware(ForumMarkRead))) // needs auth but doesn't need forums enabled
		rb.GET(hmnurl.RegexForumPost, ForumPostRedirect)
		rb.GET(hmnurl.RegexForumPostReply, needsForums(ForumPostReply))
		rb.POST(hmnurl.RegexForumPostReply, needsForums(csrfMiddleware(ForumPostReplySubmit)))
		rb.GET(hmnurl.RegexForumPostEdit, needsForums(ForumPostEdit))
		rb.POST(hmnurl.RegexForumPostEdit, needsForums(csrfMiddleware(ForumPostEditSubmit)))
		rb.GET(hmnurl.RegexForumPostDelete, needsForums(ForumPostDelete))
		rb.POST(hmnurl.RegexForumPostDelete, needsForums(csrfMiddleware(ForumPostDeleteSubmit)))
		rb.GET(hmnurl.RegexWikiArticle, WikiArticleRedirect)

		// Middleware used for blog action routes - anything related to actually creating or editing blog content
		needsBlogs := func(h Handler) Handler {
			return func(c *RequestContext) ResponseData {
				// 404 if the project has blogs disabled
				if !c.CurrentProject.HasBlog() {
					return FourOhFour(c)
				}
				// Require auth if blogs are enabled
				return needsAuth(h)(c)
			}
		}
		rb.GET(hmnurl.RegexBlog, BlogIndex)
		rb.GET(hmnurl.RegexBlogNewThread, needsBlogs(BlogNewThread))
		rb.POST(hmnurl.RegexBlogNewThread, needsBlogs(csrfMiddleware(BlogNewThreadSubmit)))
		rb.GET(hmnurl.RegexBlogThread, BlogThread)
		rb.GET(hmnurl.RegexBlogPost, BlogPostRedirectToThread)
		rb.GET(hmnurl.RegexBlogPostReply, needsBlogs(BlogPostReply))
		rb.POST(hmnurl.RegexBlogPostReply, needsBlogs(csrfMiddleware(BlogPostReplySubmit)))
		rb.GET(hmnurl.RegexBlogPostEdit, needsBlogs(BlogPostEdit))
		rb.POST(hmnurl.RegexBlogPostEdit, needsBlogs(csrfMiddleware(BlogPostEditSubmit)))
		rb.GET(hmnurl.RegexBlogPostDelete, needsBlogs(BlogPostDelete))
		rb.POST(hmnurl.RegexBlogPostDelete, needsBlogs(csrfMiddleware(BlogPostDeleteSubmit)))
		rb.GET(hmnurl.RegexBlogsRedirect, func(c *RequestContext) ResponseData {
			return c.Redirect(c.UrlContext.Url(
				fmt.Sprintf("blog%s", c.PathParams["remainder"]), nil,
			), http.StatusMovedPermanently)
		})

		rb.POST(hmnurl.RegexAssetUpload, AssetUpload)
	}
	officialProjectRoutes := anyProject.WithMiddleware(officialProjectMiddleware)
	personalProjectRoutes := hmnOnly.Group(hmnurl.RegexPersonalProject, personalProjectMiddleware)
	attachProjectRoutes(&officialProjectRoutes)
	attachProjectRoutes(&personalProjectRoutes)

	anyProject.POST(hmnurl.RegexSnippetSubmit, needsAuth(csrfMiddleware(SnippetEditSubmit)))

	anyProject.GET(hmnurl.RegexEpisodeList, EpisodeList)
	anyProject.GET(hmnurl.RegexEpisode, Episode)
	anyProject.GET(hmnurl.RegexCineraIndex, CineraIndex)

	anyProject.GET(hmnurl.RegexProjectCSS, ProjectCSS)
	anyProject.GET(hmnurl.RegexMarkdownWorkerJS, func(c *RequestContext) ResponseData {
		var res ResponseData
		res.MustWriteTemplate("markdown_worker.js", nil, c.Perf)
		res.Header().Add("Content-Type", "application/javascript")
		return res
	})

	// Other
	anyProject.AnyMethod(hmnurl.RegexCatchAll, FourOhFour)

	return router
}

func setDBConn(conn *pgxpool.Pool) Middleware {
	return func(h Handler) Handler {
		return func(c *RequestContext) ResponseData {
			c.Conn = conn
			return h(c)
		}
	}
}

func redirectToHMN(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		if !c.CurrentProject.IsHMN() {
			return c.Redirect(hmnurl.Url(c.URL().Path, hmnurl.QFromURL(c.URL())), http.StatusMovedPermanently)
		}

		return h(c)
	}
}

func officialProjectMiddleware(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		// Check if the current project (matched by subdomain) is actually no longer official
		// and therefore needs to be redirected to the personal project version of the route.
		if c.CurrentProject.Personal {
			return c.Redirect(c.UrlContext.RewriteProjectUrl(c.URL()), http.StatusSeeOther)
		}

		return h(c)
	}
}

func personalProjectMiddleware(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		hmnProject := c.CurrentProject

		id := utils.Must1(strconv.Atoi(c.PathParams["projectid"]))
		p, err := hmndata.FetchProject(c, c.Conn, c.CurrentUser, id, hmndata.ProjectsQuery{
			Lifecycles:    models.AllProjectLifecycles,
			IncludeHidden: true,
		})
		if err != nil {
			if errors.Is(err, db.NotFound) {
				return FourOhFour(c)
			} else {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch personal project"))
			}
		}

		c.CurrentProject = &p.Project
		c.CurrentProject.Color1 = hmnProject.Color1
		c.CurrentProject.Color2 = hmnProject.Color2

		c.UrlContext = hmndata.UrlContextForProject(c.CurrentProject)
		c.CurrentUserCanEditCurrentProject = CanEditProject(c.CurrentUser, p.Owners)

		if !c.CurrentProject.Personal {
			return c.Redirect(c.UrlContext.RewriteProjectUrl(c.URL()), http.StatusSeeOther)
		}

		if c.PathParams["projectslug"] != models.GeneratePersonalProjectSlug(c.CurrentProject.Name) {
			return c.Redirect(c.UrlContext.RewriteProjectUrl(c.URL()), http.StatusSeeOther)
		}

		return h(c)
	}
}

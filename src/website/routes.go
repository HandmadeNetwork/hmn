package website

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/teacat/noire"
)

func NewWebsiteRoutes(longRequestContext context.Context, conn *pgxpool.Pool) http.Handler {
	router := &Router{}
	routes := RouteBuilder{
		Router: router,
		Middleware: func(h Handler) Handler {
			return func(c *RequestContext) (res ResponseData) {
				c.Conn = conn

				logPerf := TrackRequestPerf(c)
				defer logPerf()

				defer LogContextErrorsFromResponse(c, &res)
				defer MiddlewarePanicCatcher(c, &res)

				return h(c)
			}
		},
	}

	anyProject := routes
	anyProject.Middleware = func(h Handler) Handler {
		return func(c *RequestContext) (res ResponseData) {
			c.Conn = conn

			logPerf := TrackRequestPerf(c)
			defer logPerf()

			defer LogContextErrorsFromResponse(c, &res)
			defer MiddlewarePanicCatcher(c, &res)

			defer storeNoticesInCookie(c, &res)

			ok, errRes := LoadCommonWebsiteData(c)
			if !ok {
				return errRes
			}

			return h(c)
		}
	}

	hmnOnly := routes
	hmnOnly.Middleware = func(h Handler) Handler {
		return func(c *RequestContext) (res ResponseData) {
			c.Conn = conn

			logPerf := TrackRequestPerf(c)
			defer logPerf()

			defer LogContextErrorsFromResponse(c, &res)
			defer MiddlewarePanicCatcher(c, &res)

			defer storeNoticesInCookie(c, &res)

			ok, errRes := LoadCommonWebsiteData(c)
			if !ok {
				return errRes
			}

			if !c.CurrentProject.IsHMN() {
				return c.Redirect(hmnurl.Url(c.URL().Path, hmnurl.QFromURL(c.URL())), http.StatusMovedPermanently)
			}

			return h(c)
		}
	}

	authMiddleware := func(h Handler) Handler {
		return func(c *RequestContext) (res ResponseData) {
			if c.CurrentUser == nil {
				return c.Redirect(hmnurl.BuildLoginPage(c.FullUrl()), http.StatusSeeOther)
			}

			return h(c)
		}
	}

	adminMiddleware := func(h Handler) Handler {
		return func(c *RequestContext) (res ResponseData) {
			if c.CurrentUser == nil || !c.CurrentUser.IsStaff {
				return FourOhFour(c)
			}

			return h(c)
		}
	}

	csrfMiddleware := func(h Handler) Handler {
		// CSRF mitigation actions per the OWASP cheat sheet:
		// https://cheatsheetseries.owasp.org/cheatsheets/Cross-Site_Request_Forgery_Prevention_Cheat_Sheet.html
		return func(c *RequestContext) ResponseData {
			c.Req.ParseMultipartForm(100 * 1024 * 1024)
			csrfToken := c.Req.Form.Get(auth.CSRFFieldName)
			if csrfToken != c.CurrentSession.CSRFToken {
				c.Logger.Warn().Str("userId", c.CurrentUser.Username).Msg("user failed CSRF validation - potential attack?")

				res := c.Redirect("/", http.StatusSeeOther)
				logoutUser(c, &res)

				return res
			}

			return h(c)
		}
	}

	securityTimerMiddleware := func(duration time.Duration, h Handler) Handler {
		// NOTE(asaf): Will make sure that the request takes at least `delayMs` to finish. Adds a 10% random duration.
		return func(c *RequestContext) ResponseData {
			additionalDuration := time.Duration(rand.Int63n(utils.Int64Max(1, int64(duration)/10)))
			timer := time.NewTimer(duration + additionalDuration)
			res := h(c)
			select {
			case <-longRequestContext.Done():
			case <-c.Context().Done():
			case <-timer.C:
			}
			return res
		}
	}

	routes.GET(hmnurl.RegexPublic, func(c *RequestContext) ResponseData {
		var res ResponseData
		http.StripPrefix("/public/", http.FileServer(http.Dir("public"))).ServeHTTP(&res, c.Req)
		AddCORSHeaders(c, &res)
		return res
	})

	// NOTE(asaf): HMN-only routes:
	hmnOnly.GET(hmnurl.RegexManifesto, Manifesto)
	hmnOnly.GET(hmnurl.RegexAbout, About)
	hmnOnly.GET(hmnurl.RegexCommunicationGuidelines, CommunicationGuidelines)
	hmnOnly.GET(hmnurl.RegexContactPage, ContactPage)
	hmnOnly.GET(hmnurl.RegexMonthlyUpdatePolicy, MonthlyUpdatePolicy)
	hmnOnly.GET(hmnurl.RegexProjectSubmissionGuidelines, ProjectSubmissionGuidelines)
	hmnOnly.GET(hmnurl.RegexWhenIsIt, WhenIsIt)
	hmnOnly.GET(hmnurl.RegexJamIndex, JamIndex)

	hmnOnly.GET(hmnurl.RegexOldHome, Index)

	hmnOnly.POST(hmnurl.RegexLoginAction, securityTimerMiddleware(time.Millisecond*100, Login))
	hmnOnly.GET(hmnurl.RegexLogoutAction, Logout)
	hmnOnly.GET(hmnurl.RegexLoginPage, LoginPage)

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
	hmnOnly.GET(hmnurl.RegexAdminApprovalQueue, adminMiddleware(AdminApprovalQueue))
	hmnOnly.POST(hmnurl.RegexAdminApprovalQueue, adminMiddleware(csrfMiddleware(AdminApprovalQueueSubmit)))
	hmnOnly.POST(hmnurl.RegexAdminSetUserStatus, adminMiddleware(csrfMiddleware(UserProfileAdminSetStatus)))
	hmnOnly.POST(hmnurl.RegexAdminNukeUser, adminMiddleware(csrfMiddleware(UserProfileAdminNuke)))

	hmnOnly.GET(hmnurl.RegexFeed, Feed)
	hmnOnly.GET(hmnurl.RegexAtomFeed, AtomFeed)
	hmnOnly.GET(hmnurl.RegexShowcase, Showcase)
	hmnOnly.GET(hmnurl.RegexSnippet, Snippet)
	hmnOnly.GET(hmnurl.RegexProjectIndex, ProjectIndex)

	hmnOnly.GET(hmnurl.RegexProjectNew, authMiddleware(ProjectNew))
	hmnOnly.POST(hmnurl.RegexProjectNew, authMiddleware(csrfMiddleware(ProjectNewSubmit)))

	hmnOnly.GET(hmnurl.RegexDiscordOAuthCallback, authMiddleware(DiscordOAuthCallback))
	hmnOnly.POST(hmnurl.RegexDiscordUnlink, authMiddleware(csrfMiddleware(DiscordUnlink)))
	hmnOnly.POST(hmnurl.RegexDiscordShowcaseBacklog, authMiddleware(csrfMiddleware(DiscordShowcaseBacklog)))

	hmnOnly.POST(hmnurl.RegexTwitchEventSubCallback, TwitchEventSubCallback)
	hmnOnly.GET(hmnurl.RegexTwitchDebugPage, TwitchDebugPage)

	hmnOnly.GET(hmnurl.RegexUserProfile, UserProfile)
	hmnOnly.GET(hmnurl.RegexUserSettings, authMiddleware(UserSettings))
	hmnOnly.POST(hmnurl.RegexUserSettings, authMiddleware(csrfMiddleware(UserSettingsSave)))

	hmnOnly.GET(hmnurl.RegexPodcast, PodcastIndex)
	hmnOnly.GET(hmnurl.RegexPodcastEdit, PodcastEdit)
	hmnOnly.POST(hmnurl.RegexPodcastEdit, PodcastEditSubmit)
	hmnOnly.GET(hmnurl.RegexPodcastEpisodeNew, PodcastEpisodeNew)
	hmnOnly.POST(hmnurl.RegexPodcastEpisodeNew, PodcastEpisodeSubmit)
	hmnOnly.GET(hmnurl.RegexPodcastEpisodeEdit, PodcastEpisodeEdit)
	hmnOnly.POST(hmnurl.RegexPodcastEpisodeEdit, PodcastEpisodeSubmit)
	hmnOnly.GET(hmnurl.RegexPodcastEpisode, PodcastEpisode)
	hmnOnly.GET(hmnurl.RegexPodcastRSS, PodcastRSS)

	hmnOnly.POST(hmnurl.RegexAPICheckUsername, csrfMiddleware(APICheckUsername))

	hmnOnly.GET(hmnurl.RegexLibraryAny, LibraryNotPortedYet)

	attachProjectRoutes := func(rb *RouteBuilder) {
		rb.GET(hmnurl.RegexHomepage, func(c *RequestContext) ResponseData {
			if c.CurrentProject.IsHMN() {
				return Index(c)
			} else {
				return ProjectHomepage(c)
			}
		})

		rb.GET(hmnurl.RegexProjectEdit, authMiddleware(ProjectEdit))
		rb.POST(hmnurl.RegexProjectEdit, authMiddleware(csrfMiddleware(ProjectEditSubmit)))

		// Middleware used for forum action routes - anything related to actually creating or editing forum content
		needsForums := func(h Handler) Handler {
			return func(c *RequestContext) ResponseData {
				// 404 if the project has forums disabled
				if !c.CurrentProject.HasForums() {
					return FourOhFour(c)
				}
				// Require auth if forums are enabled
				return authMiddleware(h)(c)
			}
		}
		rb.POST(hmnurl.RegexForumNewThreadSubmit, needsForums(csrfMiddleware(ForumNewThreadSubmit)))
		rb.GET(hmnurl.RegexForumNewThread, needsForums(ForumNewThread))
		rb.GET(hmnurl.RegexForumThread, ForumThread)
		rb.GET(hmnurl.RegexForum, Forum)
		rb.POST(hmnurl.RegexForumMarkRead, authMiddleware(csrfMiddleware(ForumMarkRead))) // needs auth but doesn't need forums enabled
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
				return authMiddleware(h)(c)
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
	}
	hmnOnly.Group(hmnurl.RegexPersonalProject, func(rb *RouteBuilder) {
		// TODO(ben): Perhaps someday we can make this middleware modification feel better? It seems
		// pretty common to run the outermost middleware first before doing other stuff, but having
		// to nest functions this way feels real bad.
		rb.Middleware = func(h Handler) Handler {
			return hmnOnly.Middleware(func(c *RequestContext) ResponseData {
				// At this point we are definitely on the plain old HMN subdomain.

				// Fetch personal project and do whatever
				id, err := strconv.Atoi(c.PathParams["projectid"])
				if err != nil {
					panic(oops.New(err, "project id was not numeric (bad regex in routing)"))
				}
				p, err := hmndata.FetchProject(c.Context(), c.Conn, c.CurrentUser, id, hmndata.ProjectsQuery{
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
				c.UrlContext = hmndata.UrlContextForProject(c.CurrentProject)
				c.CurrentUserCanEditCurrentProject = CanEditProject(c.CurrentUser, p.Owners)

				if !p.Project.Personal {
					return c.Redirect(c.UrlContext.RewriteProjectUrl(c.URL()), http.StatusSeeOther)
				}

				if c.PathParams["projectslug"] != models.GeneratePersonalProjectSlug(p.Project.Name) {
					return c.Redirect(c.UrlContext.RewriteProjectUrl(c.URL()), http.StatusSeeOther)
				}

				return h(c)
			})
		}
		attachProjectRoutes(rb)
	})
	anyProject.Group(regexp.MustCompile("^"), func(rb *RouteBuilder) {
		rb.Middleware = func(h Handler) Handler {
			return anyProject.Middleware(func(c *RequestContext) ResponseData {
				// We could be on any project's subdomain.

				// Check if the current project (matched by subdomain) is actually no longer official
				// and therefore needs to be redirected to the personal project version of the route.
				if c.CurrentProject.Personal {
					return c.Redirect(c.UrlContext.RewriteProjectUrl(c.URL()), http.StatusSeeOther)
				}

				return h(c)
			})
		}
		attachProjectRoutes(rb)
	})

	anyProject.POST(hmnurl.RegexAssetUpload, AssetUpload)

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

func ProjectCSS(c *RequestContext) ResponseData {
	color := c.URL().Query().Get("color")
	if color == "" {
		return c.ErrorResponse(http.StatusBadRequest, NewSafeError(nil, "You must provide a 'color' parameter.\n"))
	}

	baseData := getBaseData(c, "", nil)

	bgColor := noire.NewHex(color)
	h, s, l := bgColor.HSL()
	if baseData.Theme == "dark" {
		l = 15
	} else {
		l = 95
	}
	if s > 20 {
		s = 20
	}
	bgColor = noire.NewHSL(h, s, l)

	templateData := struct {
		templates.BaseData
		Color       string
		PostBgColor string
	}{
		BaseData:    baseData,
		Color:       color,
		PostBgColor: bgColor.HTML(),
	}

	var res ResponseData
	res.Header().Add("Content-Type", "text/css")
	err := res.WriteTemplate("project.css", templateData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to generate project CSS"))
	}

	return res
}

func FourOhFour(c *RequestContext) ResponseData {
	var res ResponseData
	res.StatusCode = http.StatusNotFound

	if c.Req.Header["Accept"] != nil && strings.Contains(c.Req.Header["Accept"][0], "text/html") {
		templateData := struct {
			templates.BaseData
			Wanted string
		}{
			BaseData: getBaseData(c, "Page not found", nil),
			Wanted:   c.FullUrl(),
		}
		res.MustWriteTemplate("404.html", templateData, c.Perf)
	} else {
		res.Write([]byte("Not Found"))
	}
	return res
}

type RejectData struct {
	templates.BaseData
	RejectReason string
}

func RejectRequest(c *RequestContext, reason string) ResponseData {
	var res ResponseData
	err := res.WriteTemplate("reject.html", RejectData{
		BaseData:     getBaseData(c, "Rejected", nil),
		RejectReason: reason,
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "Failed to render reject template"))
	}
	return res
}

func LoadCommonWebsiteData(c *RequestContext) (bool, ResponseData) {
	c.Perf.StartBlock("MIDDLEWARE", "Load common website data")
	defer c.Perf.EndBlock()

	// get user
	{
		sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
		if err == nil {
			user, session, err := getCurrentUserAndSession(c, sessionCookie.Value)
			if err != nil {
				return false, c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get current user"))
			}

			c.CurrentUser = user
			c.CurrentSession = session
		}
		// http.ErrNoCookie is the only error Cookie ever returns, so no further handling to do here.
	}

	// get official project
	{
		hostPrefix := strings.TrimSuffix(c.Req.Host, hmnurl.GetBaseHost())
		slug := strings.TrimRight(hostPrefix, ".")
		var owners []*models.User

		if len(slug) > 0 {
			dbProject, err := hmndata.FetchProjectBySlug(c.Context(), c.Conn, c.CurrentUser, slug, hmndata.ProjectsQuery{
				Lifecycles:    models.AllProjectLifecycles,
				IncludeHidden: true,
			})
			if err == nil {
				c.CurrentProject = &dbProject.Project
				c.CurrentProjectLogoUrl = templates.ProjectLogoUrl(&dbProject.Project, dbProject.LogoLightAsset, dbProject.LogoDarkAsset, c.Theme)
				owners = dbProject.Owners
			} else {
				if errors.Is(err, db.NotFound) {
					// do nothing, this is fine
				} else {
					return false, c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch current project"))
				}
			}
		}

		if c.CurrentProject == nil {
			dbProject, err := hmndata.FetchProject(c.Context(), c.Conn, c.CurrentUser, models.HMNProjectID, hmndata.ProjectsQuery{
				Lifecycles:    models.AllProjectLifecycles,
				IncludeHidden: true,
			})
			if err != nil {
				panic(oops.New(err, "failed to fetch HMN project"))
			}
			c.CurrentProject = &dbProject.Project
			c.CurrentProjectLogoUrl = templates.ProjectLogoUrl(&dbProject.Project, dbProject.LogoLightAsset, dbProject.LogoDarkAsset, c.Theme)
		}

		if c.CurrentProject == nil {
			panic("failed to load project data")
		}

		c.CurrentUserCanEditCurrentProject = CanEditProject(c.CurrentUser, owners)

		c.UrlContext = hmndata.UrlContextForProject(c.CurrentProject)
	}

	c.Theme = "light"
	if c.CurrentUser != nil && c.CurrentUser.DarkTheme {
		c.Theme = "dark"
	}

	return true, ResponseData{}
}

func AddCORSHeaders(c *RequestContext, res *ResponseData) {
	parsed, err := url.Parse(config.Config.BaseUrl)
	if err != nil {
		c.Logger.Error().Str("Config.BaseUrl", config.Config.BaseUrl).Msg("Config.BaseUrl cannot be parsed. Skipping CORS headers")
		return
	}
	origin := ""
	origins, found := c.Req.Header["Origin"]
	if found {
		origin = origins[0]
	}
	if strings.HasSuffix(origin, parsed.Host) {
		res.Header().Add("Access-Control-Allow-Origin", origin)
		res.Header().Add("Access-Control-Allow-Credentials", "true")
		res.Header().Add("Vary", "Origin")
	}
}

// Given a session id, fetches user data from the database. Will return nil if
// the user cannot be found, and will only return an error if it's serious.
func getCurrentUserAndSession(c *RequestContext, sessionId string) (*models.User, *models.Session, error) {
	session, err := auth.GetSession(c.Context(), c.Conn, sessionId)
	if err != nil {
		if errors.Is(err, auth.ErrNoSession) {
			return nil, nil, nil
		} else {
			return nil, nil, oops.New(err, "failed to get current session")
		}
	}

	user, err := db.QueryOne[models.User](c.Context(), c.Conn,
		`
		SELECT $columns{auth_user}
		FROM
			auth_user
			LEFT JOIN handmade_asset AS auth_user_avatar ON auth_user_avatar.id = auth_user.avatar_asset_id
		WHERE username = $1
		`,
		session.Username,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			logging.Debug().Str("username", session.Username).Msg("returning no current user for this request because the user for the session couldn't be found")
			return nil, nil, nil // user was deleted or something
		} else {
			return nil, nil, oops.New(err, "failed to get user for session")
		}
	}

	return user, session, nil
}

func TrackRequestPerf(c *RequestContext) (after func()) {
	c.Perf = perf.MakeNewRequestPerf(c.Route, c.Req.Method, c.Req.URL.Path)
	c.ctx = context.WithValue(c.Context(), perf.PerfContextKey, c.Perf)

	return func() {
		c.Perf.EndRequest()
		log := logging.Info()
		blockStack := make([]time.Time, 0)
		for i, block := range c.Perf.Blocks {
			for len(blockStack) > 0 && block.End.After(blockStack[len(blockStack)-1]) {
				blockStack = blockStack[:len(blockStack)-1]
			}
			log.Str(fmt.Sprintf("[%4.d] At %9.2fms", i, c.Perf.MsFromStart(&block)), fmt.Sprintf("%*.s[%s] %s (%.4fms)", len(blockStack)*2, "", block.Category, block.Description, block.DurationMs()))
			blockStack = append(blockStack, block.End)
		}
		log.Msg(fmt.Sprintf("Served [%s] %s in %.4fms", c.Perf.Method, c.Perf.Path, float64(c.Perf.End.Sub(c.Perf.Start).Nanoseconds())/1000/1000))
		// perfCollector.SubmitRun(c.Perf) // TODO(asaf): Implement a use for this
	}
}

func LogContextErrors(c *RequestContext, errs ...error) {
	for _, err := range errs {
		c.Logger.Error().Timestamp().Stack().Str("Requested", c.FullUrl()).Err(err).Msg("error occurred during request")
	}
}

func LogContextErrorsFromResponse(c *RequestContext, res *ResponseData) {
	LogContextErrors(c, res.Errors...)
}

func MiddlewarePanicCatcher(c *RequestContext, res *ResponseData) {
	if recovered := recover(); recovered != nil {
		maybeError, ok := recovered.(*error)
		var err error
		if ok {
			err = *maybeError
		} else {
			err = oops.New(nil, fmt.Sprintf("Recovered from panic with value: %v", recovered))
		}
		*res = c.ErrorResponse(http.StatusInternalServerError, err)
	}
}

const NoticesCookieName = "hmn_notices"

func getNoticesFromCookie(c *RequestContext) []templates.Notice {
	cookie, err := c.Req.Cookie(NoticesCookieName)
	if err != nil {
		if !errors.Is(err, http.ErrNoCookie) {
			c.Logger.Warn().Err(err).Msg("failed to get notices cookie")
		}
		return nil
	}
	return deserializeNoticesFromCookie(cookie.Value)
}

func storeNoticesInCookie(c *RequestContext, res *ResponseData) {
	serialized := serializeNoticesForCookie(c, res.FutureNotices)
	if serialized != "" {
		noticesCookie := http.Cookie{
			Name:     NoticesCookieName,
			Value:    serialized,
			Path:     "/",
			Domain:   config.Config.Auth.CookieDomain,
			Expires:  time.Now().Add(time.Minute * 5),
			Secure:   config.Config.Auth.CookieSecure,
			HttpOnly: true,
			SameSite: http.SameSiteLaxMode,
		}
		res.SetCookie(&noticesCookie)
	} else if !(res.StatusCode >= 300 && res.StatusCode < 400) {
		// NOTE(asaf): Don't clear on redirect
		noticesCookie := http.Cookie{
			Name:   NoticesCookieName,
			Path:   "/",
			Domain: config.Config.Auth.CookieDomain,
			MaxAge: -1,
		}
		res.SetCookie(&noticesCookie)
	}
}

func serializeNoticesForCookie(c *RequestContext, notices []templates.Notice) string {
	var builder strings.Builder
	maxSize := 1024 // NOTE(asaf): Make sure we don't use too much space for notices.
	size := 0
	for i, notice := range notices {
		sizeIncrease := len(notice.Class) + len(string(notice.Content)) + 1
		if i != 0 {
			sizeIncrease += 1
		}
		if size+sizeIncrease > maxSize {
			c.Logger.Warn().Interface("Notices", notices).Msg("Notices too big for cookie")
			break
		}

		if i != 0 {
			builder.WriteString("\t")
		}
		builder.WriteString(notice.Class)
		builder.WriteString("|")
		builder.WriteString(string(notice.Content))

		size += sizeIncrease
	}
	return builder.String()
}

func deserializeNoticesFromCookie(cookieVal string) []templates.Notice {
	var result []templates.Notice
	notices := strings.Split(cookieVal, "\t")
	for _, notice := range notices {
		parts := strings.SplitN(notice, "|", 2)
		if len(parts) == 2 {
			result = append(result, templates.Notice{
				Class:   parts[0],
				Content: template.HTML(parts[1]),
			})
		}
	}
	return result
}

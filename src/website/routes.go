package website

import (
	"context"
	"errors"
	"fmt"
	"html/template"
	"math/rand"
	"net/http"
	"net/url"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/email"
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

func NewWebsiteRoutes(longRequestContext context.Context, conn *pgxpool.Pool, perfCollector *perf.PerfCollector) http.Handler {
	router := &Router{}
	routes := RouteBuilder{
		Router: router,
		Middleware: func(h Handler) Handler {
			return func(c *RequestContext) (res ResponseData) {
				c.Conn = conn

				logPerf := TrackRequestPerf(c, perfCollector)
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

			logPerf := TrackRequestPerf(c, perfCollector)
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

			logPerf := TrackRequestPerf(c, perfCollector)
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

	anyProject.GET(hmnurl.RegexHomepage, func(c *RequestContext) ResponseData {
		if c.CurrentProject.IsHMN() {
			return Index(c)
		} else {
			return ProjectHomepage(c)
		}
	})

	// NOTE(asaf): HMN-only routes:
	hmnOnly.GET(hmnurl.RegexManifesto, Manifesto)
	hmnOnly.GET(hmnurl.RegexAbout, About)
	hmnOnly.GET(hmnurl.RegexCodeOfConduct, CodeOfConduct)
	hmnOnly.GET(hmnurl.RegexCommunicationGuidelines, CommunicationGuidelines)
	hmnOnly.GET(hmnurl.RegexContactPage, ContactPage)
	hmnOnly.GET(hmnurl.RegexMonthlyUpdatePolicy, MonthlyUpdatePolicy)
	hmnOnly.GET(hmnurl.RegexProjectSubmissionGuidelines, ProjectSubmissionGuidelines)
	hmnOnly.GET(hmnurl.RegexWhenIsIt, WhenIsIt)
	hmnOnly.GET(hmnurl.RegexJamIndex, JamIndex)

	hmnOnly.GET(hmnurl.RegexOldHome, Index)

	hmnOnly.POST(hmnurl.RegexLoginAction, securityTimerMiddleware(time.Millisecond*100, Login)) // TODO(asaf): Adjust this after launch
	hmnOnly.GET(hmnurl.RegexLogoutAction, Logout)
	hmnOnly.GET(hmnurl.RegexLoginPage, LoginPage)

	hmnOnly.GET(hmnurl.RegexRegister, RegisterNewUser)
	hmnOnly.POST(hmnurl.RegexRegister, securityTimerMiddleware(email.ExpectedEmailSendDuration, RegisterNewUserSubmit))
	hmnOnly.GET(hmnurl.RegexRegistrationSuccess, RegisterNewUserSuccess)
	hmnOnly.GET(hmnurl.RegexOldEmailConfirmation, EmailConfirmation) // TODO(asaf): Delete this a bit after launch
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

	hmnOnly.GET(hmnurl.RegexFeed, Feed)
	hmnOnly.GET(hmnurl.RegexAtomFeed, AtomFeed)
	hmnOnly.GET(hmnurl.RegexShowcase, Showcase)
	hmnOnly.GET(hmnurl.RegexSnippet, Snippet)
	hmnOnly.GET(hmnurl.RegexProjectIndex, ProjectIndex)
	hmnOnly.GET(hmnurl.RegexProjectNotApproved, ProjectHomepage)

	hmnOnly.GET(hmnurl.RegexDiscordOAuthCallback, authMiddleware(DiscordOAuthCallback))
	hmnOnly.POST(hmnurl.RegexDiscordUnlink, authMiddleware(csrfMiddleware(DiscordUnlink)))
	hmnOnly.POST(hmnurl.RegexDiscordShowcaseBacklog, authMiddleware(csrfMiddleware(DiscordShowcaseBacklog)))

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

	hmnOnly.GET(hmnurl.RegexLibraryAny, LibraryNotPortedYet)

	// NOTE(asaf): Any-project routes:
	anyProject.GET(hmnurl.RegexForumNewThread, authMiddleware(ForumNewThread))
	anyProject.POST(hmnurl.RegexForumNewThreadSubmit, authMiddleware(csrfMiddleware(ForumNewThreadSubmit)))
	anyProject.GET(hmnurl.RegexForumThread, ForumThread)
	anyProject.GET(hmnurl.RegexForum, Forum)
	anyProject.POST(hmnurl.RegexForumMarkRead, authMiddleware(csrfMiddleware(ForumMarkRead)))
	anyProject.GET(hmnurl.RegexForumPost, ForumPostRedirect)
	anyProject.GET(hmnurl.RegexForumPostReply, authMiddleware(ForumPostReply))
	anyProject.POST(hmnurl.RegexForumPostReply, authMiddleware(csrfMiddleware(ForumPostReplySubmit)))
	anyProject.GET(hmnurl.RegexForumPostEdit, authMiddleware(ForumPostEdit))
	anyProject.POST(hmnurl.RegexForumPostEdit, authMiddleware(csrfMiddleware(ForumPostEditSubmit)))
	anyProject.GET(hmnurl.RegexForumPostDelete, authMiddleware(ForumPostDelete))
	anyProject.POST(hmnurl.RegexForumPostDelete, authMiddleware(csrfMiddleware(ForumPostDeleteSubmit)))
	anyProject.GET(hmnurl.RegexWikiArticle, WikiArticleRedirect)

	anyProject.GET(hmnurl.RegexBlog, BlogIndex)
	anyProject.GET(hmnurl.RegexBlogNewThread, authMiddleware(BlogNewThread))
	anyProject.POST(hmnurl.RegexBlogNewThread, authMiddleware(csrfMiddleware(BlogNewThreadSubmit)))
	anyProject.GET(hmnurl.RegexBlogThread, BlogThread)
	anyProject.GET(hmnurl.RegexBlogPost, BlogPostRedirectToThread)
	anyProject.GET(hmnurl.RegexBlogPostReply, authMiddleware(BlogPostReply))
	anyProject.POST(hmnurl.RegexBlogPostReply, authMiddleware(csrfMiddleware(BlogPostReplySubmit)))
	anyProject.GET(hmnurl.RegexBlogPostEdit, authMiddleware(BlogPostEdit))
	anyProject.POST(hmnurl.RegexBlogPostEdit, authMiddleware(csrfMiddleware(BlogPostEditSubmit)))
	anyProject.GET(hmnurl.RegexBlogPostDelete, authMiddleware(BlogPostDelete))
	anyProject.POST(hmnurl.RegexBlogPostDelete, authMiddleware(csrfMiddleware(BlogPostDeleteSubmit)))
	anyProject.GET(hmnurl.RegexBlogsRedirect, func(c *RequestContext) ResponseData {
		return c.Redirect(hmnurl.ProjectUrl(
			fmt.Sprintf("blog%s", c.PathParams["remainder"]), nil,
			c.CurrentProject.Slug,
		), http.StatusMovedPermanently)
	})

	anyProject.POST(hmnurl.RegexAssetUpload, AssetUpload)

	anyProject.GET(hmnurl.RegexEpisodeList, EpisodeList)
	anyProject.GET(hmnurl.RegexEpisode, Episode)
	anyProject.GET(hmnurl.RegexCineraIndex, CineraIndex)

	anyProject.GET(hmnurl.RegexProjectCSS, ProjectCSS)
	anyProject.GET(hmnurl.RegexEditorPreviewsJS, func(c *RequestContext) ResponseData {
		var res ResponseData
		res.MustWriteTemplate("editorpreviews.js", nil, c.Perf)
		res.Header().Add("Content-Type", "application/javascript")
		return res
	})

	// Other
	anyProject.AnyMethod(hmnurl.RegexCatchAll, FourOhFour)

	return router
}

func FetchProjectBySlug(ctx context.Context, conn *pgxpool.Pool, slug string) (*models.Project, error) {
	if len(slug) > 0 && slug != models.HMNProjectSlug {
		subdomainProjectRow, err := db.QueryOne(ctx, conn, models.Project{}, "SELECT $columns FROM handmade_project WHERE slug = $1", slug)
		if err == nil {
			subdomainProject := subdomainProjectRow.(*models.Project)
			return subdomainProject, nil
		} else if !errors.Is(err, db.NotFound) {
			return nil, oops.New(err, "failed to get projects by slug")
		} else {
			return nil, nil
		}
	} else {
		defaultProjectRow, err := db.QueryOne(ctx, conn, models.Project{}, "SELECT $columns FROM handmade_project WHERE id = $1", models.HMNProjectID)
		if err != nil {
			if errors.Is(err, db.NotFound) {
				return nil, oops.New(nil, "default project didn't exist in the database")
			} else {
				return nil, oops.New(err, "failed to get default project")
			}
		}
		defaultProject := defaultProjectRow.(*models.Project)
		return defaultProject, nil
	}
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

	// get project
	{
		hostPrefix := strings.TrimSuffix(c.Req.Host, hmnurl.GetBaseHost())
		slug := strings.TrimRight(hostPrefix, ".")

		dbProject, err := FetchProjectBySlug(c.Context(), c.Conn, slug)
		if err != nil {
			return false, c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch current project"))
		}
		if dbProject == nil {
			return false, c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
		}

		c.CurrentProject = dbProject
	}

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

	theme := "light"
	if c.CurrentUser != nil && c.CurrentUser.DarkTheme {
		theme = "dark"
	}

	c.Theme = theme

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

	userRow, err := db.QueryOne(c.Context(), c.Conn, models.User{}, "SELECT $columns FROM auth_user WHERE username = $1", session.Username)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			logging.Debug().Str("username", session.Username).Msg("returning no current user for this request because the user for the session couldn't be found")
			return nil, nil, nil // user was deleted or something
		} else {
			return nil, nil, oops.New(err, "failed to get user for session")
		}
	}
	user := userRow.(*models.User)

	return user, session, nil
}

const PerfContextKey = "HMNPerf"

func TrackRequestPerf(c *RequestContext, perfCollector *perf.PerfCollector) (after func()) {
	c.Perf = perf.MakeNewRequestPerf(c.Route, c.Req.Method, c.Req.URL.Path)
	c.ctx = context.WithValue(c.Context(), PerfContextKey, c.Perf)

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

func ExtractPerf(ctx context.Context) *perf.RequestPerf {
	iperf := ctx.Value(PerfContextKey)
	if iperf == nil {
		return nil
	}
	return iperf.(*perf.RequestPerf)
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

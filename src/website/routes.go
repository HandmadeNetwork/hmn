package website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/teacat/noire"
)

func NewWebsiteRoutes(conn *pgxpool.Pool, perfCollector *perf.PerfCollector) http.Handler {
	router := &Router{}
	routes := RouteBuilder{
		Router: router,
		Middleware: func(h Handler) Handler {
			return func(c *RequestContext) (res ResponseData) {
				c.Conn = conn

				logPerf := TrackRequestPerf(c, perfCollector)
				defer logPerf()

				defer LogContextErrors(c, &res)

				return h(c)
			}
		},
	}

	mainRoutes := routes
	mainRoutes.Middleware = func(h Handler) Handler {
		return func(c *RequestContext) (res ResponseData) {
			c.Conn = conn

			logPerf := TrackRequestPerf(c, perfCollector)
			defer logPerf()

			defer LogContextErrors(c, &res)

			ok, errRes := LoadCommonWebsiteData(c)
			if !ok {
				return errRes
			}

			return h(c)
		}
	}

	staticPages := routes
	staticPages.Middleware = func(h Handler) Handler {
		return func(c *RequestContext) (res ResponseData) {
			c.Conn = conn

			logPerf := TrackRequestPerf(c, perfCollector)
			defer logPerf()

			defer LogContextErrors(c, &res)

			ok, errRes := LoadCommonWebsiteData(c)
			if !ok {
				return errRes
			}

			if !c.CurrentProject.IsHMN() {
				return c.Redirect(hmnurl.Url(c.URL().String(), nil), http.StatusMovedPermanently)
			}

			return h(c)
		}
	}

	// TODO(asaf): login/logout shouldn't happen on subdomains. We should verify that in the middleware.
	routes.POST(hmnurl.RegexLoginAction, Login)
	routes.GET(hmnurl.RegexLogoutAction, Logout)
	routes.StdHandler(hmnurl.RegexPublic,
		http.StripPrefix("/public/", http.FileServer(http.Dir("public"))),
	)

	mainRoutes.GET(hmnurl.RegexHomepage, func(c *RequestContext) ResponseData {
		if c.CurrentProject.IsHMN() {
			return Index(c)
		} else {
			// TODO: Return the project landing page
			panic("route not implemented")
		}
	})
	staticPages.GET(hmnurl.RegexManifesto, Manifesto)
	staticPages.GET(hmnurl.RegexAbout, About)
	staticPages.GET(hmnurl.RegexCodeOfConduct, CodeOfConduct)
	staticPages.GET(hmnurl.RegexCommunicationGuidelines, CommunicationGuidelines)
	staticPages.GET(hmnurl.RegexContactPage, ContactPage)
	staticPages.GET(hmnurl.RegexMonthlyUpdatePolicy, MonthlyUpdatePolicy)
	staticPages.GET(hmnurl.RegexProjectSubmissionGuidelines, ProjectSubmissionGuidelines)

	mainRoutes.GET(hmnurl.RegexFeed, Feed)

	// TODO(asaf): Trailing slashes break these
	mainRoutes.GET(hmnurl.RegexForumThread, ForumThread)
	// mainRoutes.GET(`^/(?P<cats>forums(/cat)*)/t/(?P<threadid>\d+)/p/(?P<postid>\d+)$`, ForumPost)
	mainRoutes.GET(hmnurl.RegexForumCategory, ForumCategory)

	mainRoutes.GET(hmnurl.RegexProjectCSS, ProjectCSS)

	mainRoutes.AnyMethod(hmnurl.RegexCatchAll, FourOhFour)

	return router
}

func getBaseData(c *RequestContext) templates.BaseData {
	var templateUser *templates.User
	if c.CurrentUser != nil {
		templateUser = &templates.User{
			Username:    c.CurrentUser.Username,
			Name:        c.CurrentUser.Name,
			Email:       c.CurrentUser.Email,
			IsSuperuser: c.CurrentUser.IsSuperuser,
			IsStaff:     c.CurrentUser.IsStaff,
		}
	}

	return templates.BaseData{
		Project:       templates.ProjectToTemplate(c.CurrentProject),
		LoginPageUrl:  hmnurl.BuildLoginPage(c.FullUrl()),
		User:          templateUser,
		Theme:         "light",
		ProjectCSSUrl: hmnurl.BuildProjectCSS(c.CurrentProject.Color1),
		Header: templates.Header{
			AdminUrl:           hmnurl.BuildHomepage(), // TODO(asaf)
			MemberSettingsUrl:  hmnurl.BuildHomepage(), // TODO(asaf)
			LoginActionUrl:     hmnurl.BuildLoginAction(c.FullUrl()),
			LogoutActionUrl:    hmnurl.BuildLogoutAction(),
			RegisterUrl:        hmnurl.BuildHomepage(), // TODO(asaf)
			HMNHomepageUrl:     hmnurl.BuildHomepage(), // TODO(asaf)
			ProjectHomepageUrl: hmnurl.BuildProjectHomepage(c.CurrentProject.Slug),
			BlogUrl:            hmnurl.BuildBlog(c.CurrentProject.Slug, 1),
			ForumsUrl:          hmnurl.BuildForumCategory(c.CurrentProject.Slug, nil, 1),
			WikiUrl:            hmnurl.BuildWiki(c.CurrentProject.Slug),
			LibraryUrl:         hmnurl.BuildLibrary(c.CurrentProject.Slug),
			ManifestoUrl:       hmnurl.BuildManifesto(),
			EpisodeGuideUrl:    hmnurl.BuildHomepage(), // TODO(asaf)
			EditUrl:            hmnurl.BuildHomepage(), // TODO(asaf)
			SearchActionUrl:    hmnurl.BuildHomepage(), // TODO(asaf)
		},
		Footer: templates.Footer{
			HomepageUrl:                hmnurl.BuildHomepage(),
			AboutUrl:                   hmnurl.BuildAbout(),
			ManifestoUrl:               hmnurl.BuildManifesto(),
			CodeOfConductUrl:           hmnurl.BuildCodeOfConduct(),
			CommunicationGuidelinesUrl: hmnurl.BuildCommunicationGuidelines(),
			ProjectIndexUrl:            hmnurl.BuildProjectIndex(),
			ForumsUrl:                  hmnurl.BuildForumCategory(models.HMNProjectSlug, nil, 1),
			ContactUrl:                 hmnurl.BuildContactPage(),
			SitemapUrl:                 hmnurl.BuildSiteMap(),
		},
	}
}

func FetchProjectBySlug(ctx context.Context, conn *pgxpool.Pool, slug string) (*models.Project, error) {
	subdomainProjectRow, err := db.QueryOne(ctx, conn, models.Project{}, "SELECT $columns FROM handmade_project WHERE slug = $1", slug)
	if err == nil {
		subdomainProject := subdomainProjectRow.(*models.Project)
		return subdomainProject, nil
	} else if !errors.Is(err, db.ErrNoMatchingRows) {
		return nil, oops.New(err, "failed to get projects by slug")
	}

	defaultProjectRow, err := db.QueryOne(ctx, conn, models.Project{}, "SELECT $columns FROM handmade_project WHERE id = $1", models.HMNProjectID)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return nil, oops.New(nil, "default project didn't exist in the database")
		} else {
			return nil, oops.New(err, "failed to get default project")
		}
	}
	defaultProject := defaultProjectRow.(*models.Project)

	return defaultProject, nil
}

func ProjectCSS(c *RequestContext) ResponseData {
	color := c.URL().Query().Get("color")
	if color == "" {
		return ErrorResponse(http.StatusBadRequest, NewSafeError(nil, "You must provide a 'color' parameter.\n"))
	}

	baseData := getBaseData(c)

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
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to generate project CSS"))
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
			BaseData: getBaseData(c),
			Wanted:   c.FullUrl(),
		}
		res.WriteTemplate("404.html", templateData, c.Perf)
	} else {
		res.Write([]byte("Not Found"))
	}
	return res
}

func LoadCommonWebsiteData(c *RequestContext) (bool, ResponseData) {
	c.Perf.StartBlock("MIDDLEWARE", "Load common website data")
	defer c.Perf.EndBlock()
	// get project
	{
		slug := ""
		hostParts := strings.SplitN(c.Req.Host, ".", 3)
		if len(hostParts) >= 3 {
			slug = hostParts[0]
		}

		dbProject, err := FetchProjectBySlug(c.Context(), c.Conn, slug)
		if err != nil {
			return false, ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch current project"))
		}

		c.CurrentProject = dbProject
	}

	{
		sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
		if err == nil {
			user, err := getCurrentUser(c, sessionCookie.Value)
			if err != nil {
				return false, ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get current user"))
			}

			c.CurrentUser = user
		}
		// http.ErrNoCookie is the only error Cookie ever returns, so no further handling to do here.
	}

	return true, ResponseData{}
}

// Given a session id, fetches user data from the database. Will return nil if
// the user cannot be found, and will only return an error if it's serious.
func getCurrentUser(c *RequestContext, sessionId string) (*models.User, error) {
	session, err := auth.GetSession(c.Context(), c.Conn, sessionId)
	if err != nil {
		if errors.Is(err, auth.ErrNoSession) {
			return nil, nil
		} else {
			return nil, oops.New(err, "failed to get current session")
		}
	}

	userRow, err := db.QueryOne(c.Context(), c.Conn, models.User{}, "SELECT $columns FROM auth_user WHERE username = $1", session.Username)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			logging.Debug().Str("username", session.Username).Msg("returning no current user for this request because the user for the session couldn't be found")
			return nil, nil // user was deleted or something
		} else {
			return nil, oops.New(err, "failed to get user for session")
		}
	}
	user := userRow.(*models.User)

	return user, nil
}

func TrackRequestPerf(c *RequestContext, perfCollector *perf.PerfCollector) (after func()) {
	c.Perf = perf.MakeNewRequestPerf(c.Route, c.Req.URL.Path)
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
		log.Msg(fmt.Sprintf("Served %s in %.4fms", c.Perf.Path, float64(c.Perf.End.Sub(c.Perf.Start).Nanoseconds())/1000/1000))
		perfCollector.SubmitRun(c.Perf)
	}
}

func LogContextErrors(c *RequestContext, res *ResponseData) {
	for _, err := range res.Errors {
		c.Logger.Error().Timestamp().Stack().Str("Requested", c.FullUrl()).Err(err).Msg("error occurred during request")
	}
}

package website

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/jackc/pgx/v4/pgxpool"
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

				defer LogContextErrors(c, res)

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

			defer LogContextErrors(c, res)

			ok, errRes := LoadCommonWebsiteData(c)
			if !ok {
				return errRes
			}

			return h(c)
		}
	}

	routes.POST("^/login$", Login)
	routes.GET("^/logout$", Logout)
	routes.StdHandler("^/public/.*$",
		http.StripPrefix("/public/", http.FileServer(http.Dir("public"))),
	)

	mainRoutes.GET("^/$", func(c *RequestContext) ResponseData {
		if c.CurrentProject.IsHMN() {
			return Index(c)
		} else {
			// TODO: Return the project landing page
			panic("route not implemented")
		}
	})
	mainRoutes.GET(`^/feed(/(?P<page>.+)?)?$`, Feed)

	mainRoutes.GET(`^/(?P<cats>forums(/.+?)*)$`, ForumCategory)
	// mainRoutes.GET(`^/(?P<cats>forums(/cat)*)/t/(?P<threadid>\d+)/p/(?P<postid>\d+)$`, ForumPost)

	mainRoutes.GET("^/assets/project.css$", ProjectCSS)

	mainRoutes.AnyMethod("", FourOhFour)

	return router
}

func getBaseData(c *RequestContext) templates.BaseData {
	var templateUser *templates.User
	if c.CurrentUser != nil {
		templateUser = &templates.User{
			Username:    c.CurrentUser.Username,
			Email:       c.CurrentUser.Email,
			IsSuperuser: c.CurrentUser.IsSuperuser,
			IsStaff:     c.CurrentUser.IsStaff,
		}
	}

	return templates.BaseData{
		Project: templates.ProjectToTemplate(c.CurrentProject),
		User:    templateUser,
		Theme:   "dark",
	}
}

func FetchProjectBySlug(ctx context.Context, conn *pgxpool.Pool, slug string) (*models.Project, error) {
	subdomainProjectRow, err := db.QueryOne(ctx, conn, models.Project{}, "SELECT $columns FROM handmade_project WHERE slug = $1", slug)
	if err == nil {
		subdomainProject := subdomainProjectRow.(models.Project)
		return &subdomainProject, nil
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

	templateData := struct {
		Color string
		Theme string
	}{
		Color: color,
		Theme: "dark",
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
	return ResponseData{
		StatusCode: http.StatusNotFound,
		Body:       bytes.NewBufferString("go away\n"),
	}
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
		for _, block := range c.Perf.Blocks {
			for len(blockStack) > 0 && block.End.After(blockStack[len(blockStack)-1]) {
				blockStack = blockStack[:len(blockStack)-1]
			}
			log.Str(fmt.Sprintf("At %9.2fms", c.Perf.MsFromStart(&block)), fmt.Sprintf("%*.s[%s] %s (%.4fms)", len(blockStack)*2, "", block.Category, block.Description, block.DurationMs()))
			blockStack = append(blockStack, block.End)
		}
		log.Msg(fmt.Sprintf("Served %s in %.4fms", c.Perf.Path, float64(c.Perf.End.Sub(c.Perf.Start).Nanoseconds())/1000/1000))
		perfCollector.SubmitRun(c.Perf)
	}
}

func LogContextErrors(c *RequestContext, res ResponseData) {
	for _, err := range res.Errors {
		c.Logger.Error().Err(err).Msg("error occurred during request")
	}
}

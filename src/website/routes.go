package website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/julienschmidt/httprouter"
)

type websiteRoutes struct {
	*HMNRouter

	conn *pgxpool.Pool
}

func NewWebsiteRoutes(conn *pgxpool.Pool) http.Handler {
	routes := &websiteRoutes{
		HMNRouter: &HMNRouter{
			HttpRouter: httprouter.New(),
			Wrappers:   []HMNHandlerWrapper{ErrorLoggingWrapper},
		},
		conn: conn,
	}

	mainRoutes := routes.WithWrappers(routes.CommonWebsiteDataWrapper)
	mainRoutes.GET("/", routes.Index)
	mainRoutes.GET("/project/:id", routes.Project)
	mainRoutes.GET("/assets/project.css", routes.ProjectCSS)

	routes.POST("/login", routes.Login)
	routes.GET("/logout", routes.Logout)

	routes.ServeFiles("/public/*filepath", http.Dir("public"))

	return routes
}

func (s *websiteRoutes) getBaseData(c *RequestContext) templates.BaseData {
	var templateUser *templates.User
	if c.currentUser != nil {
		templateUser = &templates.User{
			Username:    c.currentUser.Username,
			Email:       c.currentUser.Email,
			IsSuperuser: c.currentUser.IsSuperuser,
			IsStaff:     c.currentUser.IsStaff,
		}
	}

	return templates.BaseData{
		Project: templates.Project{
			Name:      c.currentProject.Name,
			Subdomain: c.currentProject.Slug,
			Color:     c.currentProject.Color1,

			IsHMN: c.currentProject.IsHMN(),

			HasBlog:    true,
			HasForum:   true,
			HasWiki:    true,
			HasLibrary: true,
		},
		User:  templateUser,
		Theme: "dark",
	}
}

func FetchProjectBySlug(ctx context.Context, conn *pgxpool.Pool, slug string) (*models.Project, error) {
	var subdomainProject models.Project
	err := db.QueryOneToStruct(ctx, conn, &subdomainProject, "SELECT $columns FROM handmade_project WHERE slug = $1", slug)
	if err == nil {
		return &subdomainProject, nil
	} else if !errors.Is(err, db.ErrNoMatchingRows) {
		return nil, oops.New(err, "failed to get projects by slug")
	}

	var defaultProject models.Project
	err = db.QueryOneToStruct(ctx, conn, &defaultProject, "SELECT $columns FROM handmade_project WHERE id = $1", models.HMNProjectID)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return nil, oops.New(nil, "default project didn't exist in the database")
		} else {
			return nil, oops.New(err, "failed to get default project")
		}
	}

	return &defaultProject, nil
}

func (s *websiteRoutes) Index(c *RequestContext, p httprouter.Params) {
	err := c.WriteTemplate("index.html", s.getBaseData(c))
	if err != nil {
		panic(err)
	}
}

func (s *websiteRoutes) Project(c *RequestContext, p httprouter.Params) {
	id := p.ByName("id")
	row := s.conn.QueryRow(context.Background(), "SELECT name FROM handmade_project WHERE id = $1", p.ByName("id"))
	var name string
	err := row.Scan(&name)
	if err != nil {
		panic(err)
	}

	c.Body.Write([]byte(fmt.Sprintf("(%s) %s\n", id, name)))
}

func (s *websiteRoutes) ProjectCSS(c *RequestContext, p httprouter.Params) {
	color := c.URL().Query().Get("color")
	if color == "" {
		c.StatusCode = http.StatusBadRequest
		c.Body.Write([]byte("You must provide a 'color' parameter.\n"))
		return
	}

	templateData := struct {
		Color string
		Theme string
	}{
		Color: color,
		Theme: "dark",
	}

	c.Headers().Add("Content-Type", "text/css")
	err := c.WriteTemplate("project.css", templateData)
	if err != nil {
		c.Logger.Error().Err(err).Msg("failed to generate project CSS")
		return
	}
}

func (s *websiteRoutes) Login(c *RequestContext, p httprouter.Params) {
	// TODO: Update this endpoint to give uniform responses on errors and to be resilient to timing attacks.

	form, err := c.GetFormValues()
	if err != nil {
		c.Errored(http.StatusBadRequest, NewSafeError(err, "request must contain form data"))
		return
	}

	username := form.Get("username")
	password := form.Get("password")
	if username == "" || password == "" {
		c.Errored(http.StatusBadRequest, NewSafeError(err, "you must provide both a username and password"))
	}

	redirect := form.Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	var user models.User
	err = db.QueryOneToStruct(c.Context(), s.conn, &user, "SELECT $columns FROM auth_user WHERE username = $1", username)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			c.StatusCode = http.StatusUnauthorized
		} else {
			c.Errored(http.StatusInternalServerError, oops.New(err, "failed to look up user by username"))
		}
		return
	}

	hashed, err := auth.ParsePasswordString(user.Password)
	if err != nil {
		c.Errored(http.StatusInternalServerError, oops.New(err, "failed to parse password string"))
		return
	}

	passwordsMatch, err := auth.CheckPassword(password, hashed)
	if err != nil {
		c.Errored(http.StatusInternalServerError, oops.New(err, "failed to check password against hash"))
		return
	}

	if passwordsMatch {
		session, err := auth.CreateSession(c.Context(), s.conn, username)
		if err != nil {
			c.Errored(http.StatusInternalServerError, oops.New(err, "failed to create session"))
			return
		}

		c.SetCookie(auth.NewSessionCookie(session))
		c.Redirect(redirect, http.StatusSeeOther)
		return
	} else {
		c.Redirect("/", http.StatusSeeOther) // TODO: Redirect to standalone login page with error
		return
	}
}

func (s *websiteRoutes) Logout(c *RequestContext, p httprouter.Params) {
	sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
	if err == nil {
		// clear the session from the db immediately, no expiration
		err := auth.DeleteSession(c.Context(), s.conn, sessionCookie.Value)
		if err != nil {
			logging.Error().Err(err).Msg("failed to delete session on logout")
		}
	}

	c.SetCookie(auth.DeleteSessionCookie)
	c.Redirect("/", http.StatusSeeOther) // TODO: Redirect to the page the user was currently on, or if not authorized to view that page, immediately to the home page.
}

func ErrorLoggingWrapper(h HMNHandler) HMNHandler {
	return func(c *RequestContext, p httprouter.Params) {
		h(c, p)

		for _, err := range c.Errors {
			c.Logger.Error().Err(err).Msg("error occurred during request")
		}
	}
}

func (s *websiteRoutes) CommonWebsiteDataWrapper(h HMNHandler) HMNHandler {
	return func(c *RequestContext, p httprouter.Params) {
		// get project
		{
			slug := ""
			hostParts := strings.SplitN(c.Req.Host, ".", 3)
			if len(hostParts) >= 3 {
				slug = hostParts[0]
			}

			dbProject, err := FetchProjectBySlug(c.Context(), s.conn, slug)
			if err != nil {
				c.Errored(http.StatusInternalServerError, oops.New(err, "failed to fetch current project"))
				return
			}

			c.currentProject = dbProject
		}

		sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
		if err == nil {
			user, err := s.getCurrentUserAndMember(c.Context(), sessionCookie.Value)
			if err != nil {
				c.Errored(http.StatusInternalServerError, oops.New(err, "failed to get current user and member"))
				return
			}

			c.currentUser = user
		}
		// http.ErrNoCookie is the only error Cookie ever returns, so no further handling to do here.

		h(c, p)
	}
}

// Given a session id, fetches user and member data from the database. Will return nil for
// both if neither can be found, and will only return an error if it's serious.
//
// TODO: actually return members :)
func (s *websiteRoutes) getCurrentUserAndMember(ctx context.Context, sessionId string) (*models.User, error) {
	session, err := auth.GetSession(ctx, s.conn, sessionId)
	if err != nil {
		if errors.Is(err, auth.ErrNoSession) {
			return nil, nil
		} else {
			return nil, oops.New(err, "failed to get current session")
		}
	}

	var user models.User
	err = db.QueryOneToStruct(ctx, s.conn, &user, "SELECT $columns FROM auth_user WHERE username = $1", session.Username)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			logging.Debug().Str("username", session.Username).Msg("returning no current user for this request because the user for the session couldn't be found")
			return nil, nil // user was deleted or something
		} else {
			return nil, oops.New(err, "failed to get user for session")
		}
	}

	// TODO: Also get the member model

	return &user, nil
}

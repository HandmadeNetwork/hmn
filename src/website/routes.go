package website

import (
	"bytes"
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
	mainRoutes.GET("/", func(c *RequestContext) ResponseData {
		if c.CurrentProject.ID == models.HMNProjectID {
			return routes.Index(c)
		} else {
			// TODO: Return the project landing page
			panic("route not implemented")
		}
	})
	mainRoutes.GET("/project/:id", routes.Project)
	mainRoutes.GET("/assets/project.css", routes.ProjectCSS)

	routes.POST("/login", routes.Login)
	routes.GET("/logout", routes.Logout)

	routes.ServeFiles("/public/*filepath", http.Dir("public"))

	routes.HttpRouter.NotFound = mainRoutes.WrapHandler(routes.FourOhFour)

	return routes
}

func (s *websiteRoutes) getBaseData(c *RequestContext) templates.BaseData {
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
		Project: templates.Project{
			Name:      *c.CurrentProject.Name,
			Subdomain: *c.CurrentProject.Slug,
			Color:     c.CurrentProject.Color1,

			IsHMN: c.CurrentProject.IsHMN(),

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

func (s *websiteRoutes) Project(c *RequestContext) ResponseData {
	id := c.PathParams.ByName("id")
	row := s.conn.QueryRow(context.Background(), "SELECT name FROM handmade_project WHERE id = $1", c.PathParams.ByName("id"))
	var name string
	err := row.Scan(&name)
	if err != nil {
		panic(err)
	}

	var res ResponseData
	res.Write([]byte(fmt.Sprintf("(%s) %s\n", id, name)))

	return res
}

func (s *websiteRoutes) ProjectCSS(c *RequestContext) ResponseData {
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
	res.Headers().Add("Content-Type", "text/css")
	err := res.WriteTemplate("project.css", templateData)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to generate project CSS"))
	}

	return res
}

func (s *websiteRoutes) Login(c *RequestContext) ResponseData {
	// TODO: Update this endpoint to give uniform responses on errors and to be resilient to timing attacks.

	form, err := c.GetFormValues()
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, NewSafeError(err, "request must contain form data"))
	}

	username := form.Get("username")
	password := form.Get("password")
	if username == "" || password == "" {
		return ErrorResponse(http.StatusBadRequest, NewSafeError(err, "you must provide both a username and password"))
	}

	redirect := form.Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	userRow, err := db.QueryOne(c.Context(), s.conn, models.User{}, "SELECT $columns FROM auth_user WHERE username = $1", username)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return ResponseData{
				StatusCode: http.StatusUnauthorized,
			}
		} else {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up user by username"))
		}
	}
	user := userRow.(*models.User)

	hashed, err := auth.ParsePasswordString(user.Password)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse password string"))
	}

	passwordsMatch, err := auth.CheckPassword(password, hashed)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to check password against hash"))
	}

	if passwordsMatch {
		// re-hash and save the user's password if necessary
		if hashed.IsOutdated() {
			newHashed, err := auth.HashPassword(password)
			if err == nil {
				err := auth.UpdatePassword(c.Context(), s.conn, username, newHashed)
				if err != nil {
					c.Logger.Error().Err(err).Msg("failed to update user's password")
				}
			} else {
				c.Logger.Error().Err(err).Msg("failed to re-hash password")
			}
			// If errors happen here, we can still continue with logging them in
		}

		session, err := auth.CreateSession(c.Context(), s.conn, username)
		if err != nil {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create session"))
		}

		res := c.Redirect(redirect, http.StatusSeeOther)
		res.SetCookie(auth.NewSessionCookie(session))

		return res
	} else {
		return c.Redirect("/", http.StatusSeeOther) // TODO: Redirect to standalone login page with error
	}
}

func (s *websiteRoutes) Logout(c *RequestContext) ResponseData {
	sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
	if err == nil {
		// clear the session from the db immediately, no expiration
		err := auth.DeleteSession(c.Context(), s.conn, sessionCookie.Value)
		if err != nil {
			logging.Error().Err(err).Msg("failed to delete session on logout")
		}
	}

	res := c.Redirect("/", http.StatusSeeOther) // TODO: Redirect to the page the user was currently on, or if not authorized to view that page, immediately to the home page.
	res.SetCookie(auth.DeleteSessionCookie)

	return res
}

func (s *websiteRoutes) FourOhFour(c *RequestContext) ResponseData {
	return ResponseData{
		StatusCode: http.StatusNotFound,
		Body:       bytes.NewBuffer([]byte("go away\n")),
	}
}

func ErrorLoggingWrapper(h HMNHandler) HMNHandler {
	return func(c *RequestContext) ResponseData {
		res := h(c)

		for _, err := range res.Errors {
			c.Logger.Error().Err(err).Msg("error occurred during request")
		}

		return res
	}
}

func (s *websiteRoutes) CommonWebsiteDataWrapper(h HMNHandler) HMNHandler {
	return func(c *RequestContext) ResponseData {
		// get project
		{
			slug := ""
			hostParts := strings.SplitN(c.Req.Host, ".", 3)
			if len(hostParts) >= 3 {
				slug = hostParts[0]
			}

			dbProject, err := FetchProjectBySlug(c.Context(), s.conn, slug)
			if err != nil {
				return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch current project"))
			}

			c.CurrentProject = dbProject
		}

		sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
		if err == nil {
			user, err := s.getCurrentUserAndMember(c.Context(), sessionCookie.Value)
			if err != nil {
				return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get current user and member"))
			}

			c.CurrentUser = user
		}
		// http.ErrNoCookie is the only error Cookie ever returns, so no further handling to do here.

		return h(c)
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

	userRow, err := db.QueryOne(ctx, s.conn, models.User{}, "SELECT $columns FROM auth_user WHERE username = $1", session.Username)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			logging.Debug().Str("username", session.Username).Msg("returning no current user for this request because the user for the session couldn't be found")
			return nil, nil // user was deleted or something
		} else {
			return nil, oops.New(err, "failed to get user for session")
		}
	}
	user := userRow.(*models.User)

	// TODO: Also get the member model

	return user, nil
}

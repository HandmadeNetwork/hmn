package website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
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
		HMNRouter: &HMNRouter{HttpRouter: httprouter.New()},
		conn:      conn,
	}

	mainRoutes := routes.WithWrappers(routes.CommonWebsiteDataWrapper)
	mainRoutes.GET("/", routes.Index)
	mainRoutes.GET("/project/:id", routes.Project)
	mainRoutes.GET("/assets/project.css", routes.ProjectCSS)

	routes.ServeFiles("/public/*filepath", http.Dir("public"))

	return routes
}

func (s *websiteRoutes) getBaseData(c *RequestContext) templates.BaseData {
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

func (s *websiteRoutes) CommonWebsiteDataWrapper(h HMNHandler) HMNHandler {
	return func(c *RequestContext, p httprouter.Params) {
		slug := ""
		hostParts := strings.SplitN(c.Req.Host, ".", 3)
		if len(hostParts) >= 3 {
			slug = hostParts[0]
		}

		dbProject, err := FetchProjectBySlug(c.Context(), s.conn, slug)
		if err != nil {
			c.AbortWithErrors(http.StatusInternalServerError, oops.New(err, "failed to fetch current project"))
			return
		}

		c.currentProject = dbProject

		h(c, p)
	}
}

package website

import (
	"context"
	_ "embed"
	"fmt"
	"net/http"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/templates"
	_ "git.handmade.network/hmn/hmn/src/templates"
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

	routes.GET("/", routes.Index)
	routes.GET("/project/:id", routes.Project)
	routes.GET("/assets/project.css", routes.ProjectCSS)
	routes.ServeFiles("/public/*filepath", http.Dir("public"))

	return routes
}

func (s *websiteRoutes) Index(c *RequestContext, p httprouter.Params) {
	err := c.WriteTemplate("index.html", templates.BaseData{
		Project: templates.Project{
			Name:  "Handmade Network",
			Color: "cd4e31",

			IsHMN: true,

			HasBlog:    true,
			HasForum:   true,
			HasWiki:    true,
			HasLibrary: true,
		},
		Theme: "dark",
	})
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
		logging.Error().Err(err).Msg("failed to generate project CSS")
		return
	}
}

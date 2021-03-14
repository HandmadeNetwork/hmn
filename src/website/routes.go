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
	*httprouter.Router

	conn *pgxpool.Pool
}

func NewWebsiteRoutes(conn *pgxpool.Pool) http.Handler {
	routes := &websiteRoutes{
		Router: httprouter.New(),
		conn:   conn,
	}

	routes.GET("/", routes.Index)
	routes.GET("/project/:id", routes.Project)
	routes.GET("/assets/project.css", routes.ProjectCSS)

	return routes
}

/*
TODO: Make a custom context thing so that routes won't directly use a response writer.

This should store up a body, desired headers, status codes, etc. Doing this allows us to
make middleware that can write headers after an aborted request.

This context should also provide a sub-logger with request fields so we can easily see
which URLs are having problems.
*/

func (s *websiteRoutes) Index(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	err := templates.Templates["index.html"].Execute(rw, templates.BaseData{
		ProjectColor: "cd4e31",
		Theme:        "dark",
	})
	if err != nil {
		panic(err)
	}
}

func (s *websiteRoutes) Project(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	id := p.ByName("id")
	row := s.conn.QueryRow(context.Background(), "SELECT name FROM handmade_project WHERE id = $1", p.ByName("id"))
	var name string
	err := row.Scan(&name)
	if err != nil {
		panic(err)
	}

	rw.Write([]byte(fmt.Sprintf("(%s) %s\n", id, name)))
}

func (s *websiteRoutes) ProjectCSS(rw http.ResponseWriter, r *http.Request, p httprouter.Params) {
	color := r.URL.Query().Get("color")
	if color == "" {
		rw.WriteHeader(http.StatusBadRequest)
		rw.Write([]byte("You must provide a 'color' parameter.\n"))
		return
	}

	templateData := struct {
		Color string
		Theme string
	}{
		Color: color,
		Theme: "dark",
	}

	err := templates.Templates["project.css"].Execute(rw, templateData)
	if err != nil {
		logging.Error().Err(err).Msg("failed to generate project CSS")
		return
	}
}

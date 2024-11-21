package website

import (
	"errors"
	"net/http"
	"net/url"
	"strings"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func loadCommonData(h Handler) Handler {
	return func(c *RequestContext) ResponseData {
		c.Perf.StartBlock("MIDDLEWARE", "Load common website data")
		{
			// get user
			{
				sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
				if err == nil {
					user, session, err := getCurrentUserAndSession(c, sessionCookie.Value)
					if err != nil {
						return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get current user"))
					}

					c.CurrentUser = user
					c.CurrentSession = session
				}
				// http.ErrNoCookie is the only error Cookie ever returns, so no further handling to do here.
			}

			// get current official project (HMN or otherwise, by subdomain)
			{
				slug := hmnurl.GetOfficialProjectSlugFromHost(c.Req.Host)
				var owners []*models.User

				if len(slug) > 0 {
					dbProject, err := hmndata.FetchProjectBySlug(c, c.Conn, c.CurrentUser, slug, hmndata.ProjectsQuery{
						Lifecycles:    models.AllProjectLifecycles,
						IncludeHidden: true,
					})
					if err == nil {
						c.CurrentProject = &dbProject.Project
						c.CurrentProjectLogoUrl = templates.ProjectLogoUrl(&dbProject.Project, dbProject.LogoLightAsset, dbProject.LogoDarkAsset)
						owners = dbProject.Owners
					} else {
						if errors.Is(err, db.NotFound) {
							// do nothing, this is fine
						} else {
							return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch current project"))
						}
					}
				}

				if c.CurrentProject == nil {
					dbProject, err := hmndata.FetchProject(c, c.Conn, c.CurrentUser, models.HMNProjectID, hmndata.ProjectsQuery{
						Lifecycles:    models.AllProjectLifecycles,
						IncludeHidden: true,
					})
					if err != nil {
						panic(oops.New(err, "failed to fetch HMN project"))
					}
					c.CurrentProject = &dbProject.Project
					c.CurrentProjectLogoUrl = templates.ProjectLogoUrl(&dbProject.Project, dbProject.LogoLightAsset, dbProject.LogoDarkAsset)
				}

				if c.CurrentProject == nil {
					panic("failed to load project data")
				}

				c.CurrentUserCanEditCurrentProject = CanEditProject(c.CurrentUser, owners)

				c.UrlContext = hmndata.UrlContextForProject(c.CurrentProject)
			}
		}
		c.Perf.EndBlock()

		return h(c)
	}
}

// Given a session id, fetches user data from the database. Will return nil if
// the user cannot be found, and will only return an error if it's serious.
func getCurrentUserAndSession(c *RequestContext, sessionId string) (*models.User, *models.Session, error) {
	session, err := auth.GetSession(c, c.Conn, sessionId)
	if err != nil {
		if errors.Is(err, auth.ErrNoSession) {
			return nil, nil, nil
		} else {
			return nil, nil, oops.New(err, "failed to get current session")
		}
	}

	user, err := hmndata.FetchUserByUsername(c, c.Conn, nil, session.Username, hmndata.UsersQuery{
		AnyStatus: true,
	})
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

func addCORSHeaders(c *RequestContext, res *ResponseData) {
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

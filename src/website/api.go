package website

import (
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

func APICheckUsername(c *RequestContext) ResponseData {
	c.Req.ParseForm()
	usernameArgs, hasUsername := c.Req.Form["username"]
	var user *models.User
	var err error
	if hasUsername {
		requestedUsername := usernameArgs[0]
		user, err = hmndata.FetchUserByUsername(c, c.Conn, c.CurrentUser, requestedUsername, hmndata.UsersQuery{})
		if err != nil && !errors.Is(err, db.NotFound) {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user: %s", requestedUsername))
		}
	}

	var res ResponseData
	addCORSHeaders(c, &res)
	if user != nil {
		res.WriteJson(map[string]any{
			"found":     true,
			"username":  user.Username,
			"name":      user.BestName(),
			"avatarUrl": templates.UserAvatarUrl(user),
		}, nil)
	} else {
		res.WriteJson(map[string]any{
			"found": false,
		}, nil)
	}
	return res
}

func APINewsletterSignup(c *RequestContext) ResponseData {
	input, err := c.GetJSON[struct {
		Email string `json:"email"`
	}]()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, err)
	}

	var res ResponseData

	sanitized := input.Email
	sanitized = strings.TrimSpace(sanitized)
	sanitized = strings.ToLower(sanitized)
	if len(sanitized) > 200 {
		res.StatusCode = http.StatusBadRequest
		return res
	}
	if !strings.Contains(sanitized, "@") {
		res.StatusCode = http.StatusBadRequest
		res.WriteJson(map[string]any{
			"error": "bad email",
		}, nil)
		return res
	}

	_, err = c.Conn.Exec(c,
		`
		INSERT INTO newsletter_emails (email) VALUES ($1)
		ON CONFLICT DO NOTHING
		`,
		sanitized,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save email into database"))
	}

	res.WriteHeader(http.StatusNoContent)
	return res
}

func APIProject(c *RequestContext) ResponseData {
	input, err := c.GetJSON[struct {
		Url string `json:"url"`
	}]()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, err)
	}

	if !hmnurl.UrlIsLocal(input.Url) {
		return FourOhFour(c)
	}
	parsedUrl, err := url.Parse(input.Url)
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, err)
	}

	var project hmndata.ProjectAndStuff
	if slug := hmnurl.GetOfficialProjectSlugFromHost(parsedUrl.Host); slug != "" {
		// NOTE(ben): Official project
		project, err = hmndata.FetchProjectBySlug(c, c.Conn, c.CurrentUser, slug, hmndata.ProjectsQuery{
			Lifecycles:    models.AllProjectLifecycles,
			IncludeHidden: true, // NOTE(ben): Still filtered by user visibility rules
		})
	} else if hmnurl.URLPathMatchesRoute(parsedUrl, hmnurl.RegexPersonalProject) {
		projectIDStr := hmnurl.MatchURLPathAgainstRoute(parsedUrl, hmnurl.RegexPersonalProject)["projectid"]
		projectID := utils.Must1(strconv.Atoi(projectIDStr))
		project, err = hmndata.FetchProject(c, c.Conn, c.CurrentUser, projectID, hmndata.ProjectsQuery{
			Lifecycles:    models.AllProjectLifecycles,
			IncludeHidden: true,
		})
	} else {
		return FourOhFour(c)
	}
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project through API"))
	}

	type Res struct {
		Project templates.Project `json:"project"`
		Card    string            `json:"card"`
	}
	p := templates.ProjectAndStuffToTemplate(&project)
	card := utils.Must1(templates.ExecuteTemplateByName("standalone_project_card.html", p))
	res := Res{
		Project: p,
		Card:    string(card),
	}
	return c.JSONResponse(http.StatusOK, res)
}

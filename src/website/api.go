package website

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
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
	bodyBytes := utils.Must1(io.ReadAll(c.Req.Body))
	type Input struct {
		Email string `json:"email"`
	}
	var input Input
	err := json.Unmarshal(bodyBytes, &input)
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

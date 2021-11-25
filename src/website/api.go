package website

import (
	"errors"
	"fmt"
	"net/http"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

func APICheckUsername(c *RequestContext) ResponseData {
	c.Req.ParseForm()
	usernameArgs, hasUsername := c.Req.Form["username"]
	found := false
	canonicalUsername := ""
	if hasUsername {
		requestedUsername := usernameArgs[0]
		found = true
		c.Perf.StartBlock("SQL", "Fetch user")
		userResult, err := db.QueryOne(c.Context(), c.Conn, models.User{},
			`
			SELECT $columns
			FROM
				auth_user
			WHERE
				LOWER(auth_user.username) = LOWER($1)
				AND status = ANY ($2)
			`,
			requestedUsername,
			[]models.UserStatus{models.UserStatusConfirmed, models.UserStatusApproved},
		)
		c.Perf.EndBlock()
		if err != nil {
			if errors.Is(err, db.NotFound) {
				found = false
			} else {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user: %s", requestedUsername))
			}
		} else {
			canonicalUsername = userResult.(*models.User).Username
		}
	}

	var res ResponseData
	res.Header().Set("Content-Type", "application/json")
	if found {
		res.Write([]byte(fmt.Sprintf(`{ "found": true, "canonical": "%s" }`, canonicalUsername)))
	} else {
		res.Write([]byte(`{ "found": false }`))
	}
	return res
}

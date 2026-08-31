package website

import (
	"net/http"
	"strconv"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
)

func FollowUser(c *RequestContext) ResponseData {
	err := c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse form data"))
	}

	userIDStr := c.Req.Form.Get("user_id")
	unfollowStr := c.Req.Form.Get("unfollow")

	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "failed to parse user_id field"))
	}
	unfollow := unfollowStr != ""

	if unfollow {
		_, err = c.Conn.Exec(c, `
			DELETE FROM follower
			WHERE user_id = $1 AND following_user_id = $2
		`, c.CurrentUser.ID, userID)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to unfollow user"))
		}
	} else {
		_, err = c.Conn.Exec(c, `
			INSERT INTO follower (user_id, following_user_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, c.CurrentUser.ID, userID)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to follow user"))
		}
	}

	var res ResponseData
	addCORSHeaders(c, &res)
	res.WriteHeader(http.StatusNoContent)
	return res
}

func FollowProject(c *RequestContext) ResponseData {
	err := c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse form data"))
	}

	projectIDStr := c.Req.Form.Get("project_id")
	unfollowStr := c.Req.Form.Get("unfollow")

	projectID, err := strconv.Atoi(projectIDStr)
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "failed to parse project_id field"))
	}
	unfollow := unfollowStr != ""

	if unfollow {
		_, err = c.Conn.Exec(c, `
			DELETE FROM follower
			WHERE user_id = $1 AND following_project_id = $2
		`, c.CurrentUser.ID, projectID)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to unfollow project"))
		}
	} else {
		logging.Debug().Int("userID", c.CurrentUser.ID).Int("projectID", projectID).Msg("thing")
		_, err = c.Conn.Exec(c, `
			INSERT INTO follower (user_id, following_project_id)
			VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, c.CurrentUser.ID, projectID)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to follow project"))
		}
	}

	var res ResponseData
	addCORSHeaders(c, &res)
	res.WriteHeader(http.StatusNoContent)
	return res
}

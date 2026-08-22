package website

import (
	"net/http"
	"strconv"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func FollowingTest(c *RequestContext) ResponseData {
	subforumTree := models.GetFullSubforumTree(c, c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)

	timelineItems, err := FetchFollowTimelineForUser(
		c, c.Conn,
		c.CurrentUser,
		lineageBuilder,
		FollowTimelineQuery{},
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	type FollowingTestData struct {
		templates.BaseData
		TimelineItems []templates.TimelineItem
	}

	var res ResponseData
	res.MustWriteTemplate("following_test.html", FollowingTestData{
		BaseData:      getBaseTemplateData(c, "Following test", nil),
		TimelineItems: timelineItems,
	}, c.Perf)
	return res
}

func FollowUser(c *RequestContext) ResponseData {
	err := c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse form data"))
	}

	userIDStr := c.Req.Form.Get("user_id")
	unfollowStr := c.Req.Form.Get("unfollow")
	redirect := hmnurl.SafeRedirectUrl(c.Req.Form.Get("redirect"))

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

	return c.Redirect(redirect, http.StatusSeeOther)
}

func FollowProject(c *RequestContext) ResponseData {
	err := c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse form data"))
	}

	projectIDStr := c.Req.Form.Get("project_id")
	unfollowStr := c.Req.Form.Get("unfollow")
	redirect := hmnurl.SafeRedirectUrl(c.Req.Form.Get("redirect"))

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

	return c.Redirect(redirect, http.StatusSeeOther)
}

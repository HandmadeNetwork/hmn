package website

import (
	"net/http"
	"sort"
	"strconv"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func FollowingTest(c *RequestContext) ResponseData {
	type Follower struct {
		UserID             int  `db:"user_id"`
		FollowingUserID    *int `db:"following_user_id"`
		FollowingProjectID *int `db:"following_project_id"`
	}

	following, err := db.Query[Follower](c, c.Conn, `
		SELECT $columns
		FROM follower
		WHERE user_id = $1
	`, c.CurrentUser.ID)

	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch follow data"))
	}

	projectIDs := make([]int, 0, len(following))
	userIDs := make([]int, 0, len(following))
	for _, f := range following {
		if f.FollowingProjectID != nil {
			projectIDs = append(projectIDs, *f.FollowingProjectID)
		}
		if f.FollowingUserID != nil {
			userIDs = append(userIDs, *f.FollowingUserID)
		}
	}

	var userSnippets []hmndata.SnippetAndStuff
	var projectSnippets []hmndata.SnippetAndStuff
	var userPosts []hmndata.PostAndStuff
	var projectPosts []hmndata.PostAndStuff

	if len(following) > 0 {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			ProjectIDs: projectIDs,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch projects"))
		}

		// NOTE(asaf): The original projectIDs might container hidden/abandoned projects,
		//             so we recreate it after the projects get filtered by FetchProjects.
		projectIDs = projectIDs[0:0]
		for _, p := range projects {
			projectIDs = append(projectIDs, p.Project.ID)
		}

		userSnippets, err = hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			OwnerIDs: userIDs,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user snippets"))
		}
		projectSnippets, err = hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIDs,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project snippets"))
		}

		userPosts, err = hmndata.FetchPosts(c, c.Conn, c.CurrentUser, hmndata.PostsQuery{
			UserIDs:        userIDs,
			SortDescending: true,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user posts"))
		}
		projectPosts, err = hmndata.FetchPosts(c, c.Conn, c.CurrentUser, hmndata.PostsQuery{
			ProjectIDs:     projectIDs,
			SortDescending: true,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project posts"))
		}

	}

	c.Perf.StartBlock("FOLLOWING", "Construct timeline items")
	timelineItems := make([]templates.TimelineItem, 0, len(userSnippets)+len(projectSnippets)+len(userPosts)+len(projectPosts))

	if len(userPosts) > 0 || len(projectPosts) > 0 {
		c.Perf.StartBlock("SQL", "Fetch subforum tree")
		subforumTree := models.GetFullSubforumTree(c, c.Conn)
		lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
		c.Perf.EndBlock()

		for _, post := range userPosts {
			timelineItems = append(timelineItems, PostToTimelineItem(
				hmndata.UrlContextForProject(&post.Project),
				lineageBuilder,
				&post.Post,
				&post.Thread,
				post.Author,
				c.Theme,
			))
		}
		for _, post := range projectPosts {
			timelineItems = append(timelineItems, PostToTimelineItem(
				hmndata.UrlContextForProject(&post.Project),
				lineageBuilder,
				&post.Post,
				&post.Thread,
				post.Author,
				c.Theme,
			))
		}
	}

	for _, s := range userSnippets {
		item := SnippetToTimelineItem(
			&s.Snippet,
			s.Asset,
			s.DiscordMessage,
			s.Projects,
			s.Owner,
			c.Theme,
			false,
		)
		item.SmallInfo = true
		timelineItems = append(timelineItems, item)
	}
	for _, s := range projectSnippets {
		item := SnippetToTimelineItem(
			&s.Snippet,
			s.Asset,
			s.DiscordMessage,
			s.Projects,
			s.Owner,
			c.Theme,
			false,
		)
		item.SmallInfo = true
		timelineItems = append(timelineItems, item)
	}

	// TODO(asaf): Show when they're live on twitch

	c.Perf.StartBlock("FOLLOWING", "Sort timeline")
	sort.Slice(timelineItems, func(i, j int) bool {
		return timelineItems[j].Date.Before(timelineItems[i].Date)
	})
	c.Perf.EndBlock()

	c.Perf.EndBlock()

	type FollowingTestData struct {
		templates.BaseData
		TimelineItems []templates.TimelineItem
	}

	var res ResponseData
	res.MustWriteTemplate("following_test.html", FollowingTestData{
		BaseData:      getBaseDataAutocrumb(c, "Following test"),
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

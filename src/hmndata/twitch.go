package hmndata

import (
	"context"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
)

const InvalidUserTwitchID = "INVALID_USER"

type TwitchStreamer struct {
	TwitchID    string
	TwitchLogin string
	UserID      *int
	ProjectID   *int
}

var twitchRegex = regexp.MustCompile(`twitch\.tv/(?P<login>[^/]+)$`)

type TwitchStreamersQuery struct {
	UserIDs    []int
	ProjectIDs []int
}

func FetchTwitchStreamers(ctx context.Context, dbConn db.ConnOrTx, q TwitchStreamersQuery) ([]TwitchStreamer, error) {
	defer perf.StartBlock(ctx, "TWITCH", "Fetch Twitch streamers").End()

	var qb db.QueryBuilder
	qb.Add(
		`
		---- Fetch Twitch links
		SELECT $columns{link}
		FROM
			link
			LEFT JOIN hmn_user AS link_owner ON link_owner.id = link.user_id
		WHERE
			TRUE
		`,
	)
	if len(q.UserIDs) > 0 && len(q.ProjectIDs) > 0 {
		qb.Add(
			`AND (link.user_id = ANY ($?) OR link.project_id = ANY ($?))`,
			q.UserIDs,
			q.ProjectIDs,
		)
	} else {
		if len(q.UserIDs) > 0 {
			qb.Add(`AND link.user_id = ANY ($?)`, q.UserIDs)
		}
		if len(q.ProjectIDs) > 0 {
			qb.Add(`AND link.project_id = ANY ($?)`, q.ProjectIDs)
		}
	}

	qb.Add(
		`
			AND url ~* 'twitch\.tv/([^/]+)$'
			AND ((link.user_id IS NOT NULL AND link_owner.status = $?) OR (link.project_id IS NOT NULL AND
			(SELECT COUNT(*)
			FROM
				user_project AS hup
				JOIN hmn_user AS project_owner ON project_owner.id = hup.user_id
			WHERE
				hup.project_id = link.project_id AND
				project_owner.status != $?
			) = 0))
		`,
		models.UserStatusApproved,
		models.UserStatusApproved,
	)
	dbStreamers, err := db.Query[models.Link](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch twitch links")
	}

	result := make([]TwitchStreamer, 0, len(dbStreamers))
	for _, dbStreamer := range dbStreamers {
		streamer := TwitchStreamer{
			UserID:    dbStreamer.UserID,
			ProjectID: dbStreamer.ProjectID,
		}

		match := twitchRegex.FindStringSubmatch(dbStreamer.URL)
		if match != nil {
			login := strings.ToLower(match[twitchRegex.SubexpIndex("login")])
			streamer.TwitchLogin = login
		}
		if len(streamer.TwitchLogin) > 0 {
			duplicate := false
			for _, r := range result {
				if r.TwitchLogin == streamer.TwitchLogin {
					duplicate = true
					break
				}
			}
			if !duplicate {
				result = append(result, streamer)
			}
		}
	}

	return result, nil
}

func FetchTwitchLoginsForUserOrProject(ctx context.Context, dbConn db.ConnOrTx, userId *int, projectId *int) ([]string, error) {
	defer perf.StartBlock(ctx, "TWITCH", "Fetch Twitch logins").End()

	links, err := db.Query[models.Link](ctx, dbConn,
		`
		---- Fetch Twitch links
		SELECT $columns
		FROM
			link
		WHERE
			url ~* 'twitch\.tv/([^/]+)$'
			AND ((user_id = $1 AND project_id IS NULL) OR (user_id IS NULL AND project_id = $2))
		ORDER BY url ASC
		`,
		userId,
		projectId,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch twitch links")
	}
	result := make([]string, 0, len(links))

	for _, l := range links {
		match := twitchRegex.FindStringSubmatch(l.URL)
		if match != nil {
			login := strings.ToLower(match[twitchRegex.SubexpIndex("login")])
			result = append(result, login)
		}
	}
	return result, nil
}

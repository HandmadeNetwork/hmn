package hmndata

import (
	"context"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

const InvalidUserTwitchID = "INVALID_USER"

type TwitchStreamer struct {
	TwitchID    string
	TwitchLogin string
	UserID      *int
	ProjectID   *int
}

var twitchRegex = regexp.MustCompile(`twitch\.tv/(?P<login>[^/]+)$`)

func FetchTwitchStreamers(ctx context.Context, dbConn db.ConnOrTx) ([]TwitchStreamer, error) {
	type linkResult struct {
		Link models.Link `db:"link"`
	}
	streamers, err := db.Query(ctx, dbConn, linkResult{},
		`
		SELECT $columns
		FROM
			handmade_links AS link
			LEFT JOIN auth_user AS link_owner ON link_owner.id = link.user_id
		WHERE
			url ~* 'twitch\.tv/([^/]+)$' AND
			((link.user_id IS NOT NULL AND link_owner.status = $1) OR (link.project_id IS NOT NULL AND
			(SELECT COUNT(*)
			FROM
				handmade_user_projects AS hup
				JOIN auth_user AS project_owner ON project_owner.id = hup.user_id
			WHERE
				hup.project_id = link.project_id AND
				project_owner.status != $1
			) = 0))
		`,
		models.UserStatusApproved,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch twitch links")
	}

	result := make([]TwitchStreamer, 0, len(streamers))
	for _, s := range streamers {
		dbStreamer := s.(*linkResult).Link

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
	links, err := db.Query(ctx, dbConn, models.Link{},
		`
		SELECT $columns
		FROM
			handmade_links AS link
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
		url := l.(*models.Link).URL
		match := twitchRegex.FindStringSubmatch(url)
		if match != nil {
			login := strings.ToLower(match[twitchRegex.SubexpIndex("login")])
			result = append(result, login)
		}
	}
	return result, nil
}

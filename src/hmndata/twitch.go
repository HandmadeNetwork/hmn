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
	streamers, err := db.Query(ctx, dbConn, models.Link{},
		`
		SELECT $columns
		FROM
			handmade_links AS link
		WHERE
			url ~* 'twitch\.tv/([^/]+)$'
		`,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch twitch links")
	}

	result := make([]TwitchStreamer, 0, len(streamers))
	for _, s := range streamers {
		dbStreamer := s.(*models.Link)

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

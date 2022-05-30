package twitch

import (
	"context"
	"encoding/json"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"github.com/jackc/pgx/v4/pgxpool"
)

type twitchNotification struct {
	Status streamStatus
	Type   twitchNotificationType
}

var twitchNotificationChannel chan twitchNotification
var linksChangedChannel chan struct{}

func MonitorTwitchSubscriptions(ctx context.Context, dbConn *pgxpool.Pool) jobs.Job {
	log := logging.ExtractLogger(ctx).With().Str("twitch goroutine", "stream monitor").Logger()
	ctx = logging.AttachLoggerToContext(&log, ctx)

	if config.Config.Twitch.ClientID == "" {
		log.Warn().Msg("No twitch config provided.")
		return jobs.Noop()
	}

	twitchNotificationChannel = make(chan twitchNotification, 100)
	linksChangedChannel = make(chan struct{}, 10)
	job := jobs.New()

	go func() {
		defer func() {
			log.Info().Msg("Shutting down twitch monitor")
			job.Done()
		}()
		log.Info().Msg("Running twitch monitor...")

		err := refreshAccessToken(ctx)
		if err != nil {
			log.Error().Err(err).Msg("Failed to fetch refresh token on start")
			return
		}

		monitorTicker := time.NewTicker(2 * time.Hour)
		firstRunChannel := make(chan struct{}, 1)
		firstRunChannel <- struct{}{}

		timers := make([]*time.Timer, 0)
		expiredTimers := make(chan *time.Timer, 10)
		for {
			select {
			case <-ctx.Done():
				for _, timer := range timers {
					timer.Stop()
				}
				return
			case expired := <-expiredTimers:
				for idx, timer := range timers {
					if timer == expired {
						timers = append(timers[:idx], timers[idx+1:]...)
						break
					}
				}
			case <-firstRunChannel:
				syncWithTwitch(ctx, dbConn, true)
			case <-monitorTicker.C:
				syncWithTwitch(ctx, dbConn, true)
			case <-linksChangedChannel:
				// NOTE(asaf): Since we update links inside transactions for users/projects
				//             we won't see the updated list of links until the transaction is committed.
				//             Waiting 5 seconds is just a quick workaround for that. It's not
				//             convenient to only trigger this after the transaction is committed.
				var timer *time.Timer
				t := time.AfterFunc(5*time.Second, func() {
					expiredTimers <- timer
					syncWithTwitch(ctx, dbConn, false)
				})
				timer = t
				timers = append(timers, t)
			case notification := <-twitchNotificationChannel:
				if notification.Type == notificationTypeRevocation {
					syncWithTwitch(ctx, dbConn, false)
				} else {
					// NOTE(asaf): The twitch API (getStreamStatus) lags behind the notification and
					//             would return old data if we called it immediately, so we process
					//             the notification to the extent we can, and later do a full update. We can get the
					//             category from the notification, but not the tags (or the up-to-date title),
					//             so we can't really skip this.
					var timer *time.Timer
					t := time.AfterFunc(3*time.Minute, func() {
						expiredTimers <- timer
						updateStreamStatus(ctx, dbConn, notification.Status.TwitchID, notification.Status.TwitchLogin)
					})
					timer = t
					timers = append(timers, t)
					processEventSubNotification(ctx, dbConn, &notification)
				}
			}
		}
	}()

	return job
}

type twitchNotificationType int

const (
	notificationTypeNone          twitchNotificationType = 0
	notificationTypeOnline                               = 1
	notificationTypeOffline                              = 2
	notificationTypeChannelUpdate                        = 3

	notificationTypeRevocation = 4
)

func QueueTwitchNotification(messageType string, body []byte) error {
	var notification twitchNotification
	if messageType == "notification" {
		type notificationJson struct {
			Subscription struct {
				Type string `json:"type"`
			} `json:"subscription"`
			Event struct {
				BroadcasterUserID    string `json:"broadcaster_user_id"`
				BroadcasterUserLogin string `json:"broadcaster_user_login"`
				Title                string `json:"title"`
				CategoryID           string `json:"category_id"`
			} `json:"event"`
		}
		var incoming notificationJson
		err := json.Unmarshal(body, &incoming)
		if err != nil {
			return oops.New(err, "failed to parse notification body")
		}

		notification.Status.TwitchID = incoming.Event.BroadcasterUserID
		notification.Status.TwitchLogin = incoming.Event.BroadcasterUserLogin
		notification.Status.Title = incoming.Event.Title
		notification.Status.Category = incoming.Event.CategoryID
		notification.Status.StartedAt = time.Now()
		switch incoming.Subscription.Type {
		case "stream.online":
			notification.Type = notificationTypeOnline
			notification.Status.Live = true
		case "stream.offline":
			notification.Type = notificationTypeOffline
		case "channel.update":
			notification.Type = notificationTypeChannelUpdate
			// NOTE(asaf): Can't tell if the user is live here.
		default:
			return oops.New(nil, "unknown subscription type received")
		}
	} else if messageType == "revocation" {
		notification.Type = notificationTypeRevocation
	}

	if twitchNotificationChannel != nil && notification.Type != notificationTypeNone {
		select {
		case twitchNotificationChannel <- notification:
		default:
			return oops.New(nil, "twitch notification channel is full")
		}
	}
	return nil
}

func UserOrProjectLinksUpdated(twitchLoginsPreChange, twitchLoginsPostChange []string) {
	if linksChangedChannel != nil {
		twitchChanged := (len(twitchLoginsPreChange) != len(twitchLoginsPostChange))
		if !twitchChanged {
			for idx, _ := range twitchLoginsPreChange {
				if twitchLoginsPreChange[idx] != twitchLoginsPostChange[idx] {
					twitchChanged = true
					break
				}
			}
		}
		select {
		case linksChangedChannel <- struct{}{}:
		default:
		}
	}
}

func syncWithTwitch(ctx context.Context, dbConn *pgxpool.Pool, updateAll bool) {
	log := logging.ExtractLogger(ctx)
	log.Info().Msg("Running twitch sync")
	p := perf.MakeNewRequestPerf("Background job", "", "syncWithTwitch")
	defer func() {
		p.EndRequest()
		perf.LogPerf(p, log.Info())
	}()

	type twitchSyncStats struct {
		NumSubbed         int
		NumUnsubbed       int
		NumStreamsChecked int
	}
	var stats twitchSyncStats

	p.StartBlock("SQL", "Fetch list of streamers")
	streamers, err := hmndata.FetchTwitchStreamers(ctx, dbConn)
	if err != nil {
		log.Error().Err(err).Msg("Error while monitoring twitch")
		return
	}
	p.EndBlock()

	needID := make([]string, 0)
	streamerMap := make(map[string]*hmndata.TwitchStreamer)
	for idx, streamer := range streamers {
		needID = append(needID, streamer.TwitchLogin)
		streamerMap[streamer.TwitchLogin] = &streamers[idx]
	}

	p.StartBlock("TwitchAPI", "Fetch twitch user info")
	twitchUsers, err := getTwitchUsersByLogin(ctx, needID)
	if err != nil {
		log.Error().Err(err).Msg("Error while monitoring twitch")
		return
	}
	p.EndBlock()

	for _, tu := range twitchUsers {
		streamerMap[tu.TwitchLogin].TwitchID = tu.TwitchID
	}

	validStreamers := make([]hmndata.TwitchStreamer, 0, len(streamers))
	for _, streamer := range streamers {
		if len(streamer.TwitchID) > 0 {
			validStreamers = append(validStreamers, streamer)
		}
	}

	p.StartBlock("TwitchAPI", "Fetch event subscriptions")
	subscriptions, err := getEventSubscriptions(ctx)
	if err != nil {
		log.Error().Err(err).Msg("Error while monitoring twitch")
		return
	}
	p.EndBlock()

	const (
		EventSubNone    = 0 // No event of this type found
		EventSubRefresh = 1 // Event found, but bad status. Need to unsubscribe and resubscribe.
		EventSubGood    = 2 // All is well.
	)

	type isSubbedByType map[string]bool

	streamerEventSubs := make(map[string]isSubbedByType)
	for _, streamer := range validStreamers {
		streamerEventSubs[streamer.TwitchID] = make(isSubbedByType)
		streamerEventSubs[streamer.TwitchID]["channel.update"] = false
		streamerEventSubs[streamer.TwitchID]["stream.online"] = false
		streamerEventSubs[streamer.TwitchID]["stream.offline"] = false
	}

	type unsubEvent struct {
		TwitchID string
		EventID  string
	}

	toUnsub := make([]unsubEvent, 0)

	for _, sub := range subscriptions {
		handled := false
		if eventSubs, ok := streamerEventSubs[sub.TwitchID]; ok {
			if _, ok := eventSubs[sub.Type]; ok { // Make sure it's a known type
				if !sub.GoodStatus {
					log.Debug().Str("TwitchID", sub.TwitchID).Str("Event Type", sub.Type).Msg("Twitch doesn't like our sub")
					toUnsub = append(toUnsub, unsubEvent{TwitchID: sub.TwitchID, EventID: sub.EventID})
				} else {
					streamerEventSubs[sub.TwitchID][sub.Type] = true
				}
				handled = true
			}
		}
		if !handled {
			// NOTE(asaf): Found an unknown type or an event subscription that we don't have a matching user for.
			//             Make sure we unsubscribe.
			toUnsub = append(toUnsub, unsubEvent{TwitchID: sub.TwitchID, EventID: sub.EventID})
		}
	}

	if config.Config.Env != config.Dev { // NOTE(asaf): Can't subscribe to events from dev. We need a non-localhost callback url.
		p.StartBlock("TwitchAPI", "Sync subscriptions with twitch")
		for _, ev := range toUnsub {
			err = unsubscribeFromEvent(ctx, ev.EventID)
			if err != nil {
				log.Error().Err(err).Msg("Error while unsubscribing events")
				// NOTE(asaf): Soft error. Don't care if it fails.
			}
			stats.NumUnsubbed += 1
		}

		for twitchID, evStatuses := range streamerEventSubs {
			for evType, isSubbed := range evStatuses {
				if !isSubbed {
					err = subscribeToEvent(ctx, evType, twitchID)
					if err != nil {
						log.Error().Err(err).Msg("Error while monitoring twitch")
						return
					}
					stats.NumSubbed += 1
				}
			}
		}
		p.EndBlock()
	}

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to start transaction")
	}
	defer tx.Rollback(ctx)

	allIDs := make([]string, 0, len(validStreamers))
	for _, streamer := range validStreamers {
		allIDs = append(allIDs, streamer.TwitchID)
	}
	p.StartBlock("SQL", "Remove untracked streamers")
	_, err = tx.Exec(ctx,
		`DELETE FROM twitch_stream WHERE twitch_id != ANY($1)`,
		allIDs,
	)
	if err != nil {
		log.Error().Err(err).Msg("Failed to remove untracked twitch ids from streamer list in db")
		return
	}
	p.EndBlock()

	usersToUpdate := make([]string, 0)
	if updateAll {
		usersToUpdate = allIDs
	} else {
		// NOTE(asaf): Twitch can revoke our subscriptions, so we need to
		//             update users whose subs were revoked or missing since last time we checked.
		for twitchID, evStatuses := range streamerEventSubs {
			for _, isSubbed := range evStatuses {
				if !isSubbed {
					usersToUpdate = append(usersToUpdate, twitchID)
					break
				}
			}
		}
	}

	p.StartBlock("TwitchAPI", "Fetch twitch stream statuses")
	statuses, err := getStreamStatus(ctx, usersToUpdate)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch stream statuses")
		return
	}
	p.EndBlock()
	p.StartBlock("SQL", "Update stream statuses in db")
	for _, status := range statuses {
		log.Debug().Interface("Status", status).Msg("Got streamer")
		err = updateStreamStatusInDB(ctx, tx, &status)
		if err != nil {
			log.Error().Err(err).Msg("failed to update twitch stream status")
		}
	}
	p.EndBlock()
	err = tx.Commit(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to commit transaction")
	}
	stats.NumStreamsChecked += len(usersToUpdate)
	log.Info().Interface("Stats", stats).Msg("Twitch sync done")

	log.Debug().Msg("Notifying discord")
	err = notifyDiscordOfLiveStream(ctx, dbConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to notify discord")
	}
}

func notifyDiscordOfLiveStream(ctx context.Context, dbConn db.ConnOrTx) error {
	streams, err := db.Query[models.TwitchStream](ctx, dbConn,
		`
		SELECT $columns
		FROM
			twitch_stream
		ORDER BY started_at ASC
		`,
	)
	if err != nil {
		return oops.New(err, "failed to fetch livestreams from db")
	}

	var streamDetails []hmndata.StreamDetails
	for _, s := range streams {
		streamDetails = append(streamDetails, hmndata.StreamDetails{
			Username:  s.Login,
			StartTime: s.StartedAt,
			Title:     s.Title,
		})
	}

	err = discord.UpdateStreamers(ctx, dbConn, streamDetails)
	if err != nil {
		return oops.New(err, "failed to update discord with livestream info")
	}
	return nil
}

func updateStreamStatus(ctx context.Context, dbConn db.ConnOrTx, twitchID string, twitchLogin string) {
	log := logging.ExtractLogger(ctx)
	log.Debug().Str("TwitchID", twitchID).Msg("Updating stream status")
	var err error

	// NOTE(asaf): Verifying that the streamer we're processing hasn't been removed from our db in the meantime.
	foundStreamer := false
	allStreamers, err := hmndata.FetchTwitchStreamers(ctx, dbConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch hmn streamers")
		return
	}
	for _, streamer := range allStreamers {
		if streamer.TwitchLogin == twitchLogin {
			foundStreamer = true
			break
		}
	}
	if !foundStreamer {
		return
	}

	status := streamStatus{
		TwitchID: twitchID,
		Live:     false,
	}

	result, err := getStreamStatus(ctx, []string{twitchID})
	if err != nil {
		log.Error().Str("TwitchID", twitchID).Err(err).Msg("failed to fetch stream status")
		return
	}
	if len(result) > 0 {
		log.Debug().Interface("Got status", result[0]).Msg("Got streamer status from twitch")
		status = result[0]
	}
	err = updateStreamStatusInDB(ctx, dbConn, &status)
	if err != nil {
		log.Error().Err(err).Msg("failed to update twitch stream status")
		return
	}

	log.Debug().Msg("Notifying discord")
	err = notifyDiscordOfLiveStream(ctx, dbConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to notify discord")
	}
}

func processEventSubNotification(ctx context.Context, dbConn db.ConnOrTx, notification *twitchNotification) {
	log := logging.ExtractLogger(ctx)
	log.Debug().Interface("Notification", notification).Msg("Processing twitch notification")
	if notification.Type == notificationTypeNone {
		return
	}

	// NOTE(asaf): Verifying that the streamer we're processing hasn't been removed from our db in the meantime.
	foundStreamer := false
	allStreamers, err := hmndata.FetchTwitchStreamers(ctx, dbConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch hmn streamers")
		return
	}
	for _, streamer := range allStreamers {
		if streamer.TwitchLogin == notification.Status.TwitchLogin {
			foundStreamer = true
			break
		}
	}
	if !foundStreamer {
		return
	}

	if notification.Type == notificationTypeOnline || notification.Type == notificationTypeOffline {
		log.Debug().Interface("Status", notification.Status).Msg("Updating status")
		err = updateStreamStatusInDB(ctx, dbConn, &notification.Status)
		if err != nil {
			log.Error().Err(err).Msg("failed to update twitch stream status")
			return
		}
	} else if notification.Type == notificationTypeChannelUpdate {
		// NOTE(asaf): Channel updates can happen wether or not the streamer is live.
		//             So we just update the title if the user is live in our db, and we rely on the
		//             3-minute delayed status update to verify live status and category/tag requirements.
		_, err = dbConn.Exec(ctx,
			`
			UPDATE twitch_stream
			SET title = $1
			WHERE twitch_id = $2
			`,
			notification.Status.Title,
			notification.Status.TwitchID,
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to update twitch stream status")
			return
		}
	}

	log.Debug().Msg("Notifying discord")
	err = notifyDiscordOfLiveStream(ctx, dbConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to notify discord")
	}
}

func updateStreamStatusInDB(ctx context.Context, conn db.ConnOrTx, status *streamStatus) error {
	log := logging.ExtractLogger(ctx)
	if isStatusRelevant(status) {
		log.Debug().Msg("Status relevant")
		_, err := conn.Exec(ctx,
			`
			INSERT INTO twitch_stream (twitch_id, twitch_login, title, started_at)
				VALUES ($1, $2, $3, $4)
			ON CONFLICT (twitch_id) DO UPDATE SET
				title = EXCLUDED.title,
				started_at = EXCLUDED.started_at
			`,
			status.TwitchID,
			status.TwitchLogin,
			status.Title,
			status.StartedAt,
		)
		if err != nil {
			return oops.New(err, "failed to insert twitch streamer into db")
		}
	} else {
		log.Debug().Msg("Stream not relevant")
		_, err := conn.Exec(ctx,
			`
			DELETE FROM twitch_stream WHERE twitch_id = $1
			`,
			status.TwitchID,
		)
		if err != nil {
			return oops.New(err, "failed to remove twitch streamer from db")
		}
	}
	return nil
}

var RelevantCategories = []string{
	"1469308723", // Software and Game Development
}

var RelevantTags = []string{
	"a59f1e4e-257b-4bd0-90c7-189c3efbf917", // Programming
	"6f86127d-6051-4a38-94bb-f7b475dde109", // Software Development
}

func isStatusRelevant(status *streamStatus) bool {
	if status.Live {
		for _, cat := range RelevantCategories {
			if status.Category == cat {
				return true
			}
		}

		for _, tag := range RelevantTags {
			for _, streamTag := range status.Tags {
				if tag == streamTag {
					return true
				}
			}
		}
	}
	return false
}

package twitch

import (
	"context"
	"encoding/json"
	"fmt"
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
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v4/pgxpool"
)

// NOTE(asaf): The twitch api madness:
//
//               | stream.online | stream.offline | channel.update[3] | REST[1][2]
// id[4]         |    YES        |     NO         |     NO            |    YES
// twitch_id     |    YES        |     YES        |     YES           |    YES
// twitch_login  |    YES        |     YES        |     YES           |    YES
// is_live       |    YES        |   IMPLICIT     |     NO            |    YES
// started_at    |    YES        |     NO         |     NO            |    YES
// title         |    NO         |     NO         |     YES           |    YES
// cat_id        |    NO         |     NO         |     YES           |    YES
// tags          |    NO         |     NO         |     NO            |    YES
//
// [1] REST returns nothing when user is not live
// [2] Information received from REST is ~3 minutes old.
// [3] channel.update also fires when the user changes their twitch channel settings when they're not live (i.e. as soon as they update it in twitch settings)
// [4] ID of the current livestream

type streamStatus struct {
	StreamID    string
	TwitchID    string
	TwitchLogin string
	Live        bool
	Title       string
	StartedAt   time.Time
	CategoryID  string
	Tags        []string
}

type twitchNotificationType int

const (
	notificationTypeNone          twitchNotificationType = 0
	notificationTypeOnline                               = 1
	notificationTypeOffline                              = 2
	notificationTypeChannelUpdate                        = 3

	notificationTypeRevocation = 4
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

		monitorTicker := time.NewTicker(2 * time.Hour)
		firstRunChannel := make(chan struct{}, 1)
		firstRunChannel <- struct{}{}

		timers := make([]*time.Timer, 0)
		expiredTimers := make(chan *time.Timer, 10)
		for {
			done, err := func() (done bool, retErr error) {
				defer utils.RecoverPanicAsError(&retErr)
				select {
				case <-ctx.Done():
					for _, timer := range timers {
						timer.Stop()
					}
					return true, nil
				case expired := <-expiredTimers:
					for idx, timer := range timers {
						if timer == expired {
							timers = append(timers[:idx], timers[idx+1:]...)
							break
						}
					}
				case <-firstRunChannel:
					err := refreshAccessToken(ctx)
					if err != nil {
						log.Error().Err(err).Msg("Failed to fetch refresh token on start")
						return true, nil
					}
					syncWithTwitch(ctx, dbConn, true, true)
				case <-monitorTicker.C:
					twitchLogClear(ctx, dbConn)
					syncWithTwitch(ctx, dbConn, true, true)
				case <-linksChangedChannel:
					// NOTE(asaf): Since we update links inside transactions for users/projects
					//             we won't see the updated list of links until the transaction is committed.
					//             Waiting 5 seconds is just a quick workaround for that. It's not
					//             convenient to only trigger this after the transaction is committed.
					var timer *time.Timer
					t := time.AfterFunc(5*time.Second, func() {
						expiredTimers <- timer
						syncWithTwitch(ctx, dbConn, false, false)
					})
					timer = t
					timers = append(timers, t)
				case notification := <-twitchNotificationChannel:
					if notification.Type == notificationTypeRevocation {
						syncWithTwitch(ctx, dbConn, false, false)
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
				return false, nil
			}()
			if err != nil {
				log.Error().Err(err).Msg("Panicked in MonitorTwitchSubscriptions")
			} else if done {
				return
			}
		}
	}()

	return job
}

func QueueTwitchNotification(ctx context.Context, conn db.ConnOrTx, messageType string, body []byte) error {
	var notification twitchNotification
	if messageType == "notification" {
		type notificationJson struct {
			Subscription struct {
				Type string `json:"type"`
			} `json:"subscription"`
			Event struct {
				StreamID             string `json:"id"`
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

		twitchLog(ctx, conn, models.TwitchLogTypeHook, incoming.Event.BroadcasterUserLogin, "Got hook: "+incoming.Subscription.Type, string(body))

		notification.Status.TwitchID = incoming.Event.BroadcasterUserID
		notification.Status.TwitchLogin = incoming.Event.BroadcasterUserLogin
		notification.Status.Title = incoming.Event.Title
		notification.Status.CategoryID = incoming.Event.CategoryID
		notification.Status.StartedAt = time.Now()
		switch incoming.Subscription.Type {
		case "stream.online":
			notification.Type = notificationTypeOnline
			notification.Status.Live = true
			notification.Status.StreamID = incoming.Event.StreamID
		case "stream.offline":
			notification.Type = notificationTypeOffline
		case "channel.update":
			notification.Type = notificationTypeChannelUpdate
			// NOTE(asaf): Can't tell if the user is live here.
		default:
			return oops.New(nil, "unknown subscription type received")
		}
	} else if messageType == "revocation" {
		twitchLog(ctx, conn, models.TwitchLogTypeHook, "", "Got hook: Revocation", string(body))
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

func syncWithTwitch(ctx context.Context, dbConn *pgxpool.Pool, updateAll bool, updateVODs bool) {
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
		`DELETE FROM twitch_latest_status WHERE twitch_id != ANY($1)`,
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
	twitchLog(ctx, tx, models.TwitchLogTypeOther, "", "Batch resync", fmt.Sprintf("%#v", statuses))
	p.EndBlock()
	p.StartBlock("SQL", "Update stream statuses in db")
	for _, twitchId := range usersToUpdate {
		var status *streamStatus
		for idx, st := range statuses {
			if st.TwitchID == twitchId {
				status = &statuses[idx]
				break
			}
		}
		if status == nil {
			twitchLogin := ""
			for _, streamer := range validStreamers {
				if streamer.TwitchID == twitchId {
					twitchLogin = streamer.TwitchLogin
					break
				}
			}
			status = &streamStatus{
				TwitchID:    twitchId,
				TwitchLogin: twitchLogin,
			}
		}
		twitchLog(ctx, tx, models.TwitchLogTypeREST, status.TwitchLogin, "Resync", fmt.Sprintf("%#v", status))
		err = gotRESTUpdate(ctx, tx, status)
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

	if updateVODs {
		err = findMissingVODs(ctx, dbConn)
		if err != nil {
			log.Error().Err(err).Msg("failed to find missing twitch vods")
		}
		err = verifyHistoryVODs(ctx, dbConn)
		if err != nil {
			log.Error().Err(err).Msg("failed to verify twitch vods")
		}
	}

	log.Debug().Msg("Notifying discord")
	err = notifyDiscordOfLiveStream(ctx, dbConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to notify discord")
	}
}

func notifyDiscordOfLiveStream(ctx context.Context, dbConn db.ConnOrTx) error {
	history, err := db.Query[models.TwitchStreamHistory](ctx, dbConn,
		`
		SELECT $columns
		FROM twitch_stream_history
		WHERE discord_needs_update = TRUE
		ORDER BY started_at ASC
		`,
	)
	if err != nil {
		return oops.New(err, "failed to fetch twitch history")
	}

	updatedHistories := make([]*models.TwitchStreamHistory, 0)
	for _, h := range history {
		relevant := isStreamRelevant(h.CategoryID, h.Tags)
		if relevant && !h.EndedAt.IsZero() {
			msgId, err := discord.PostStreamHistory(ctx, h)
			if err != nil {
				return oops.New(err, "failed to post twitch history to discord")
			}
			h.DiscordNeedsUpdate = false
			h.DiscordMessageID = msgId
			updatedHistories = append(updatedHistories, h)
		}
	}

	for _, h := range updatedHistories {
		_, err = dbConn.Exec(ctx,
			`
			UPDATE twitch_stream_history
			SET
				discord_needs_update = $2,
				discord_message_id = $3,
			WHERE stream_id = $1
			`,
			h.StreamID,
			h.DiscordNeedsUpdate,
			h.DiscordMessageID,
		)
		if err != nil {
			return oops.New(err, "failed to update twitch history after posting to discord")
		}
	}

	streams, err := db.Query[models.TwitchLatestStatus](ctx, dbConn,
		`
		SELECT $columns
		FROM
			twitch_latest_status
		WHERE live = TRUE
		ORDER BY started_at ASC
		`,
	)
	if err != nil {
		return oops.New(err, "failed to fetch livestreams from db")
	}

	var streamDetails []hmndata.StreamDetails
	for _, s := range streams {
		if isStreamRelevant(s.CategoryID, s.Tags) {
			streamDetails = append(streamDetails, hmndata.StreamDetails{
				Username:  s.TwitchLogin,
				StartTime: s.StartedAt,
				Title:     s.Title,
			})
		}
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
		TwitchID:    twitchID,
		TwitchLogin: twitchLogin,
		Live:        false,
	}

	result, err := getStreamStatus(ctx, []string{twitchID})
	if err != nil {
		log.Error().Str("TwitchID", twitchID).Err(err).Msg("failed to fetch stream status")
		return
	}
	twitchLog(ctx, dbConn, models.TwitchLogTypeREST, twitchLogin, "Fetched status", fmt.Sprintf("%#v", result))
	if len(result) > 0 {
		log.Debug().Interface("Got status", result[0]).Msg("Got streamer status from twitch")
		status = result[0]
	}
	err = gotRESTUpdate(ctx, dbConn, &status)
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

	twitchLog(ctx, dbConn, models.TwitchLogTypeHook, notification.Status.TwitchLogin, "Processing hook", fmt.Sprintf("%#v", notification))
	switch notification.Type {
	case notificationTypeOnline:
		err := gotStreamOnline(ctx, dbConn, &notification.Status)
		if err != nil {
			log.Error().Err(err).Msg("failed to update twitch stream status")
		}
	case notificationTypeOffline:
		err := gotStreamOffline(ctx, dbConn, &notification.Status)
		if err != nil {
			log.Error().Err(err).Msg("failed to update twitch stream status")
		}
	case notificationTypeChannelUpdate:
		err := gotChannelUpdate(ctx, dbConn, &notification.Status)
		if err != nil {
			log.Error().Err(err).Msg("failed to update twitch stream status")
		}
	}

	log.Debug().Msg("Notifying discord")
	err = notifyDiscordOfLiveStream(ctx, dbConn)
	if err != nil {
		log.Error().Err(err).Msg("failed to notify discord")
	}
}

func gotStreamOnline(ctx context.Context, conn db.ConnOrTx, status *streamStatus) error {
	latest, err := fetchLatestStreamStatus(ctx, conn, status.TwitchID, status.TwitchLogin)
	if err != nil {
		return err
	}
	latest.Live = true
	latest.StreamID = status.StreamID
	latest.StartedAt = status.StartedAt
	latest.LastHookLiveUpdate = time.Now()
	err = saveLatestStreamStatus(ctx, conn, latest)
	if err != nil {
		return err
	}
	err = updateStreamHistory(ctx, conn, latest)
	if err != nil {
		return err
	}
	return nil
}

func gotStreamOffline(ctx context.Context, conn db.ConnOrTx, status *streamStatus) error {
	latest, err := fetchLatestStreamStatus(ctx, conn, status.TwitchID, status.TwitchLogin)
	if err != nil {
		return err
	}
	latest.Live = false
	latest.LastHookLiveUpdate = time.Now()
	err = saveLatestStreamStatus(ctx, conn, latest)
	if err != nil {
		return err
	}
	err = updateStreamHistory(ctx, conn, latest)
	if err != nil {
		return err
	}
	return nil
}

func gotChannelUpdate(ctx context.Context, conn db.ConnOrTx, status *streamStatus) error {
	latest, err := fetchLatestStreamStatus(ctx, conn, status.TwitchID, status.TwitchLogin)
	if err != nil {
		return err
	}
	if !latest.Live {
		// NOTE(asaf): If the stream is live, this channel update applies
		//             to the current livestream. Otherwise, this will
		//             only apply to the next stream, so we clear out
		//             the stream info.
		latest.StreamID = ""
		latest.StartedAt = time.Time{}
	}
	latest.Title = status.Title
	if latest.CategoryID != status.CategoryID {
		latest.CategoryID = status.CategoryID
		latest.Tags = []string{} // NOTE(asaf): We don't get tags here, but we can't assume they didn't change because some tags are automatic based on category
	}
	latest.LastHookChannelUpdate = time.Now()
	err = saveLatestStreamStatus(ctx, conn, latest)
	if err != nil {
		return err
	}
	err = updateStreamHistory(ctx, conn, latest)
	if err != nil {
		return err
	}
	return nil
}

func gotRESTUpdate(ctx context.Context, conn db.ConnOrTx, status *streamStatus) error {
	latest, err := fetchLatestStreamStatus(ctx, conn, status.TwitchID, status.TwitchLogin)
	if err != nil {
		return err
	}
	if latest.LastHookLiveUpdate.Add(3 * time.Minute).Before(time.Now()) {
		latest.Live = status.Live
		if status.Live {
			// NOTE(asaf): We don't get this information if the user isn't live
			latest.StartedAt = status.StartedAt
			latest.StreamID = status.StreamID
		}
	}
	if latest.LastHookChannelUpdate.Add(3 * time.Minute).Before(time.Now()) {
		if status.Live {
			// NOTE(asaf): We don't get this information if the user isn't live
			latest.Title = status.Title
			latest.CategoryID = status.CategoryID
			latest.Tags = status.Tags
		}
	}
	latest.LastRESTUpdate = time.Now()
	err = saveLatestStreamStatus(ctx, conn, latest)
	if err != nil {
		return err
	}
	err = updateStreamHistory(ctx, conn, latest)
	if err != nil {
		return err
	}
	return nil
}

func fetchLatestStreamStatus(ctx context.Context, conn db.ConnOrTx, twitchID string, twitchLogin string) (*models.TwitchLatestStatus, error) {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to begin transaction for stream status fetch")
	}
	defer tx.Rollback(ctx)

	result, err := db.QueryOne[models.TwitchLatestStatus](ctx, conn,
		`
		SELECT $columns
		FROM twitch_latest_status
		WHERE twitch_id = $1
		`,
		twitchID,
	)
	if err == db.NotFound {
		_, err = conn.Exec(ctx,
			`
			INSERT INTO twitch_latest_status (twitch_id, twitch_login)
			VALUES ($1, $2)
			`,
			twitchID,
			twitchLogin,
		)
		if err != nil {
			return nil, err
		}
		result = &models.TwitchLatestStatus{
			TwitchID:    twitchID,
			TwitchLogin: twitchLogin,
		}
	} else if err != nil {
		return nil, oops.New(err, "failed to fetch existing twitch status")
	}

	if result.TwitchLogin != twitchLogin {
		_, err = conn.Exec(ctx,
			`
			UPDATE twitch_latest_status
			SET twitch_login = $2
			WHERE twitch_id = $1
			`,
			twitchID,
			twitchLogin,
		)
		if err != nil {
			return nil, oops.New(err, "failed to update twitch login")
		}
		result.TwitchLogin = twitchLogin
	}
	err = tx.Commit(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to commit transaction")
	}
	return result, nil
}

func saveLatestStreamStatus(ctx context.Context, conn db.ConnOrTx, latest *models.TwitchLatestStatus) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return oops.New(err, "failed to start transaction for stream status update")
	}
	defer tx.Rollback(ctx)

	// NOTE(asaf): Ensure that we have a record for it in the db
	_, err = fetchLatestStreamStatus(ctx, conn, latest.TwitchID, latest.TwitchLogin)
	if err != nil {
		return err
	}

	_, err = conn.Exec(ctx,
		`
		UPDATE twitch_latest_status
		SET
			live = $2,
			started_at = $3,
			title = $4,
			category_id = $5,
			tags = $6,
			last_hook_live_update = $7,
			last_hook_channel_update = $8,
			last_rest_update = $9
		WHERE
			twitch_id = $1
		`,
		latest.TwitchID,
		latest.Live,
		latest.StartedAt,
		latest.Title,
		latest.CategoryID,
		latest.Tags,
		latest.LastHookLiveUpdate,
		latest.LastHookChannelUpdate,
		latest.LastRESTUpdate,
	)
	if err != nil {
		return oops.New(err, "failed to update twitch latest status")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return oops.New(err, "failed to commit transaction")
	}
	return nil
}

func updateStreamHistory(ctx context.Context, dbConn db.ConnOrTx, status *models.TwitchLatestStatus) error {
	if status.StreamID == "" {
		return nil
	}
	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return oops.New(err, "failed to begin transaction for stream history update")
	}
	defer tx.Rollback(ctx)
	history, err := db.QueryOne[models.TwitchStreamHistory](ctx, tx,
		`
		SELECT $columns
		FROM twitch_stream_history
		WHERE stream_id = $1
		`,
		status.StreamID,
	)

	if err == db.NotFound {
		history = &models.TwitchStreamHistory{}
		history.StreamID = status.StreamID
		history.TwitchID = status.TwitchID
		history.TwitchLogin = status.TwitchLogin
		history.StartedAt = status.StartedAt
		history.DiscordNeedsUpdate = true
	} else if err != nil {
		return oops.New(err, "failed to fetch existing stream history")
	}

	if !status.Live && history.EndedAt.IsZero() {
		history.EndedAt = time.Now()
		history.EndApproximated = true
		history.DiscordNeedsUpdate = true
	}

	history.Title = status.Title
	history.CategoryID = status.CategoryID
	history.Tags = status.Tags

	_, err = tx.Exec(ctx,
		`
		INSERT INTO
		twitch_stream_history (stream_id, twitch_id, twitch_login, started_at, ended_at, title, category_id, tags)
		VALUES                ($1,        $2,        $3,           $4,         $5,       $6,    $7,          $8)
		ON CONFLICT (stream_id) DO UPDATE SET
			ended_at = EXCLUDED.ended_at,
			title = EXCLUDED.title,
			category_id = EXCLUDED.category_id,
			tags = EXCLUDED.tags
		`,
		history.StreamID,
		history.TwitchID,
		history.TwitchLogin,
		history.StartedAt,
		history.EndedAt,
		history.Title,
		history.CategoryID,
		history.Tags,
	)
	if err != nil {
		return oops.New(err, "failed to insert/update twitch history")
	}
	err = tx.Commit(ctx)
	if err != nil {
		return oops.New(err, "failed to commit transaction")
	}

	if !history.EndedAt.IsZero() {
		err = findHistoryVOD(ctx, dbConn, history)
		if err != nil {
			return oops.New(err, "failed to look up twitch vod")
		}
	}
	return nil
}

func findHistoryVOD(ctx context.Context, dbConn db.ConnOrTx, history *models.TwitchStreamHistory) error {
	if history.StreamID == "" || history.VODID != "" || history.VODGone {
		return nil
	}

	vods, err := getArchivedVideosForUser(ctx, history.TwitchID, 10)
	if err != nil {
		return oops.New(err, "failed to fetch vods for streamer")
	}

	var vod *archivedVideo
	for idx, v := range vods {
		if v.StreamID == history.StreamID {
			vod = &vods[idx]
		}
	}
	if vod != nil {
		history.VODID = vod.ID
		history.VODUrl = vod.VODUrl
		history.VODThumbnail = vod.VODThumbnail
		history.LastVerifiedVOD = time.Now()
		history.VODGone = false

		if vod.Duration.Minutes() > 0 {
			history.EndedAt = history.StartedAt.Add(vod.Duration)
			history.EndApproximated = false
		}

		_, err = dbConn.Exec(ctx,
			`
			UPDATE twitch_stream_history
			SET
				vod_id = $2,
				vod_url = $3,
				vod_thumbnail = $4,
				last_verified_vod = $5,
				vod_gone = $6,
				ended_at = $7,
				end_approximated = $8
			WHERE stream_id = $1
			`,
			history.StreamID,
			history.VODID,
			history.VODUrl,
			history.VODThumbnail,
			history.LastVerifiedVOD,
			history.VODGone,
			history.EndedAt,
			history.EndApproximated,
		)
		if err != nil {
			return oops.New(err, "failed to update stream history with VOD")
		}
	} else {
		if history.StartedAt.Add(14 * 24 * time.Hour).Before(time.Now()) {
			history.VODGone = true
			_, err = dbConn.Exec(ctx, `
				UPDATE twitch_stream_history
				SET
					vod_gone = $2,
				WHERE stream_id = $1
				`,
				history.StreamID,
				history.VODGone,
			)
			if err != nil {
				return oops.New(err, "failed to update stream history")
			}
		}
	}
	return nil
}

func findMissingVODs(ctx context.Context, dbConn db.ConnOrTx) error {
	histories, err := db.Query[models.TwitchStreamHistory](ctx, dbConn,
		`
		SELECT $columns
		FROM twitch_stream_history
		WHERE
			vod_gone = FALSE AND
			vod_url = '' AND
			ended_at != $1
		`,
		time.Time{},
	)
	if err != nil {
		return oops.New(err, "failed to fetch stream history for vod updates")
	}

	for _, history := range histories {
		err = findHistoryVOD(ctx, dbConn, history)
		if err != nil {
			return err
		}
	}
	return nil
}

func verifyHistoryVODs(ctx context.Context, dbConn db.ConnOrTx) error {
	histories, err := db.Query[models.TwitchStreamHistory](ctx, dbConn,
		`
		SELECT $columns
		FROM twitch_stream_history
		WHERE
			vod_gone = FALSE AND
			vod_id != '' AND 
			last_verified_vod < $1
		ORDER BY last_verified_vod ASC
		LIMIT 100
		`,
		time.Now().Add(-24*time.Hour),
	)

	if err != nil {
		return oops.New(err, "failed to fetch stream history for vod verification")
	}

	videoIDs := make([]string, 0, len(histories))
	for _, h := range histories {
		videoIDs = append(videoIDs, h.VODID)
	}

	VODs, err := getArchivedVideos(ctx, videoIDs)
	if err != nil {
		return oops.New(err, "failed to fetch vods from twitch")
	}

	vodGone := make([]string, 0, len(histories))
	vodFound := make([]string, 0, len(histories))
	for _, h := range histories {
		found := false
		for _, vod := range VODs {
			if h.VODID == vod.ID {
				found = true
				break
			}
		}
		if !found {
			vodGone = append(vodGone, h.VODID)
		} else {
			vodFound = append(vodFound, h.VODID)
		}
	}

	if len(vodGone) > 0 {
		_, err = dbConn.Exec(ctx,
			`
			UPDATE twitch_stream_history
			SET
				discord_needs_update = TRUE,
				vod_id = '',
				vod_url = '',
				vod_thumbnail = '',
				last_verified_vod = $2,
				vod_gone = TRUE
			WHERE
				vod_id = ANY($1)
			`,
			vodGone,
			time.Now(),
		)
		if err != nil {
			return oops.New(err, "failed to update twitch history")
		}
	}

	if len(vodFound) > 0 {
		_, err = dbConn.Exec(ctx,
			`
			UPDATE twitch_stream_history
			SET
				last_verified_vod = $2,
			WHERE
				vod_id = ANY($1)
			`,
			vodFound,
			time.Now(),
		)
		if err != nil {
			return oops.New(err, "failed to update twitch history")
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

func isStreamRelevant(catID string, tags []string) bool {
	for _, cat := range RelevantCategories {
		if cat == catID {
			return true
		}
	}

	for _, tag := range RelevantTags {
		for _, streamTag := range tags {
			if tag == streamTag {
				return true
			}
		}
	}
	return false
}

func twitchLog(ctx context.Context, conn db.ConnOrTx, logType models.TwitchLogType, login string, message string, payload string) {
	_, err := conn.Exec(ctx,
		`
		INSERT INTO twitch_log (logged_at, twitch_login, type, message, payload)
		VALUES ($1, $2, $3, $4, $5)
		`,
		time.Now(),
		login,
		logType,
		message,
		payload,
	)
	if err != nil {
		log := logging.ExtractLogger(ctx).With().Str("twitch goroutine", "twitch logger").Logger()
		log.Error().Err(err).Msg("Failed to log twitch event")
	}
}

func twitchLogClear(ctx context.Context, conn db.ConnOrTx) {
	_, err := conn.Exec(ctx,
		`
		DELETE FROM twitch_log
		WHERE timestamp <= $1
		`,
		time.Now().Add(-(time.Hour * 24 * 4)),
	)
	if err != nil {
		log := logging.ExtractLogger(ctx).With().Str("twitch goroutine", "twitch logger").Logger()
		log.Error().Err(err).Msg("Failed to clear old twitch logs")
	}
}

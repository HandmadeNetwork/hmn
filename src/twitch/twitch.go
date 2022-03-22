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
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"github.com/jackc/pgx/v4/pgxpool"
)

type twitchNotification struct {
	TwitchID string
	Type     twitchNotificationType
}

var twitchNotificationChannel chan twitchNotification
var linksChangedChannel chan struct{}

func MonitorTwitchSubscriptions(ctx context.Context, dbConn *pgxpool.Pool) <-chan struct{} {
	log := logging.ExtractLogger(ctx).With().Str("twitch goroutine", "stream monitor").Logger()
	ctx = logging.AttachLoggerToContext(&log, ctx)

	if config.Config.Twitch.ClientID == "" {
		log.Warn().Msg("No twitch config provided.")
		done := make(chan struct{}, 1)
		done <- struct{}{}
		return done
	}

	twitchNotificationChannel = make(chan twitchNotification, 100)
	linksChangedChannel = make(chan struct{}, 10)
	done := make(chan struct{})

	go func() {
		defer func() {
			log.Info().Msg("Shutting down twitch monitor")
			done <- struct{}{}
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
		for {
			select {
			case <-ctx.Done():
				return
			case <-firstRunChannel:
				syncWithTwitch(ctx, dbConn, true)
			case <-monitorTicker.C:
				syncWithTwitch(ctx, dbConn, true)
			case <-linksChangedChannel:
				syncWithTwitch(ctx, dbConn, false)
			case notification := <-twitchNotificationChannel:
				if notification.Type == notificationTypeRevocation {
					syncWithTwitch(ctx, dbConn, false)
				} else {
					processEventSubNotification(ctx, dbConn, &notification)
				}
			}
		}
	}()

	return done
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
			} `json:"event"`
		}
		var incoming notificationJson
		err := json.Unmarshal(body, &incoming)
		if err != nil {
			return oops.New(err, "failed to parse notification body")
		}

		notification.TwitchID = incoming.Event.BroadcasterUserID
		switch incoming.Subscription.Type {
		case "stream.online":
			notification.Type = notificationTypeOnline
		case "stream.offline":
			notification.Type = notificationTypeOffline
		case "channel.update":
			notification.Type = notificationTypeChannelUpdate
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
		if twitchChanged {
			// NOTE(asaf): Since we update links inside transactions for users/projects
			//             we won't see the updated list of links until the transaction is committed.
			//             Waiting 10 seconds is just a quick workaround for that. It's not
			//             convenient to only trigger this after the transaction is committed.
			time.AfterFunc(10*time.Second, func() {
				linksChangedChannel <- struct{}{}
			})
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
					eventSubs[sub.Type] = true
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
		`DELETE FROM twitch_streams WHERE twitch_id != ANY($1)`,
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
		_, err = updateStreamStatusInDB(ctx, tx, &status)
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
}

func notifyDiscordOfLiveStream(ctx context.Context, dbConn db.ConnOrTx, twitchLogin string) error {
	var err error
	if config.Config.Discord.StreamsChannelID != "" {
		err = discord.SendMessages(ctx, dbConn, discord.MessageToSend{
			ChannelID: config.Config.Discord.StreamsChannelID,
			Req: discord.CreateMessageRequest{
				Content: fmt.Sprintf("%s is live: https://twitch.tv/%s", twitchLogin, twitchLogin),
			},
		})
	}
	return err
}

func processEventSubNotification(ctx context.Context, dbConn db.ConnOrTx, notification *twitchNotification) {
	if notification.Type == notificationTypeNone {
		return
	}

	log := logging.ExtractLogger(ctx)
	status := streamStatus{
		TwitchID: notification.TwitchID,
		Live:     false,
	}
	var err error
	if notification.Type == notificationTypeChannelUpdate || notification.Type == notificationTypeOnline {
		result, err := getStreamStatus(ctx, []string{notification.TwitchID})
		if err != nil || len(result) == 0 {
			log.Error().Str("TwitchID", notification.TwitchID).Err(err).Msg("failed to fetch stream status")
			return
		}
		allStreamers, err := hmndata.FetchTwitchStreamers(ctx, dbConn)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch hmn streamers")
			return
		}
		for _, streamer := range allStreamers {
			if streamer.TwitchLogin == result[0].TwitchLogin {
				status = result[0]
				break
			}
		}
	}

	inserted, err := updateStreamStatusInDB(ctx, dbConn, &status)
	if err != nil {
		log.Error().Err(err).Msg("failed to update twitch stream status")
	}
	if inserted && notification.Type == notificationTypeOnline {
		err = notifyDiscordOfLiveStream(ctx, dbConn, status.TwitchLogin)
		if err != nil {
			log.Error().Err(err).Msg("failed to notify discord")
		}
	}
}

func updateStreamStatusInDB(ctx context.Context, conn db.ConnOrTx, status *streamStatus) (bool, error) {
	inserted := false
	if isStatusRelevant(status) {
		_, err := conn.Exec(ctx,
			`
			INSERT INTO twitch_streams (twitch_id, twitch_login, title, started_at)
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
			return false, oops.New(err, "failed to insert twitch streamer into db")
		}
		inserted = true
	} else {
		_, err := conn.Exec(ctx,
			`
			DELETE FROM twitch_streams WHERE twitch_id = $1
			`,
			status.TwitchID,
		)
		if err != nil {
			return false, oops.New(err, "failed to remove twitch streamer from db")
		}
		inserted = false
	}
	return inserted, nil
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

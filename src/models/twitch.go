package models

import "time"

type TwitchLatestStatus struct {
	TwitchID              string    `db:"twitch_id"`
	TwitchLogin           string    `db:"twitch_login"`
	StreamID              string    `db:"stream_id"`
	Live                  bool      `db:"live"`
	StartedAt             time.Time `db:"started_at"`
	Title                 string    `db:"title"`
	CategoryID            string    `db:"category_id"`
	Tags                  []string  `db:"tags"`
	LastHookLiveUpdate    time.Time `db:"last_hook_live_update"`
	LastHookChannelUpdate time.Time `db:"last_hook_channel_update"`
	LastRESTUpdate        time.Time `db:"last_rest_update"`
}

type TwitchStreamHistory struct {
	StreamID           string    `db:"stream_id"`
	TwitchID           string    `db:"twitch_id"`
	TwitchLogin        string    `db:"twitch_login"`
	StartedAt          time.Time `db:"started_at"`
	EndedAt            time.Time `db:"ended_at"`
	StreamEnded        bool      `db:"stream_ended"`
	EndApproximated    bool      `db:"end_approximated"`
	Title              string    `db:"title"`
	CategoryID         string    `db:"category_id"`
	Tags               []string  `db:"tags"`
	DiscordMessageID   string    `db:"discord_message_id"`
	DiscordNeedsUpdate bool      `db:"discord_needs_update"`
	VODID              string    `db:"vod_id"`
	VODUrl             string    `db:"vod_url"`
	VODThumbnail       string    `db:"vod_thumbnail"`
	LastVerifiedVOD    time.Time `db:"last_verified_vod"`
	// NOTE(asaf): If we had a VOD for a while, and then it disappeared,
	//             assume it was removed from twitch and don't bother
	//             checking for it again.
	VODGone bool `db:"vod_gone"`
}

type TwitchLogType int

const (
	TwitchLogTypeOther TwitchLogType = iota + 1
	TwitchLogTypeHook
	TwitchLogTypeREST
)

type TwitchLog struct {
	ID       int           `db:"id"`
	LoggedAt time.Time     `db:"logged_at"`
	Login    string        `db:"twitch_login"`
	Type     TwitchLogType `db:"type"`
	Message  string        `db:"message"`
	Payload  string        `db:"payload"`
}

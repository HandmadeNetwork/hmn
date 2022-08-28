package models

import "time"

type TwitchID struct {
	ID    string `db:"id"`
	Login string `db:"login"`
}

type TwitchStream struct {
	ID        string    `db:"twitch_id"`
	Login     string    `db:"twitch_login"`
	Title     string    `db:"title"`
	StartedAt time.Time `db:"started_at"`
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

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

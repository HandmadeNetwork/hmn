package models

import (
	"time"
)

type DiscordSession struct {
	ID             string `db:"session_id"`
	SequenceNumber int    `db:"sequence_number"`
}

type DiscordOutgoingMessage struct {
	ID          int       `db:"id"`
	ChannelID   string    `db:"channel_id"`
	PayloadJSON string    `db:"payload_json"`
	ExpiresAt   time.Time `db:"expires_at"`
}

type DiscordMessage struct {
	ID             string    `db:"id"`
	ChannelID      string    `db:"channel_id"`
	GuildID        *string   `db:"guild_id"`
	Url            string    `db:"url"`
	UserID         string    `db:"user_id"`
	SentAt         time.Time `db:"sent_at"`
	SnippetCreated bool      `db:"snippet_created"`
}

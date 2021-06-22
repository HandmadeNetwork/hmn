package models

import (
	"time"
)

type DiscordMessage struct {
	ID             string    `db:"id"`
	ChannelID      string    `db:"channel_id"`
	GuildID        *string   `db:"guild_id"`
	Url            string    `db:"url"`
	UserID         string    `db:"user_id"`
	SentAt         time.Time `db:"sent_at"`
	SnippetCreated bool      `db:"snippet_created"`
}

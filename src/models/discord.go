package models

import (
	"time"

	"github.com/google/uuid"
)

type DiscordUser struct {
	ID            int       `db:"id"`
	Username      string    `db:"username"`
	Discriminator string    `db:"discriminator"`
	AccessToken   string    `db:"access_token"`
	RefreshToken  string    `db:"refresh_token"`
	Avatar        *string   `db:"avatar"`
	Locale        string    `db:"locale"`
	UserID        string    `db:"userid"`
	Expiry        time.Time `db:"expiry"`
	HMNUserId     int       `db:"hmn_user_id"`
}

/*
Logs the existence of a Discord message and what we've done with it.
Created unconditionally for all users, regardless of link status.
Therefore, it must not contain any actual content.
*/
type DiscordMessage struct {
	ID             string    `db:"id"`
	ChannelID      string    `db:"channel_id"`
	GuildID        *string   `db:"guild_id"`
	Url            string    `db:"url"`
	UserID         string    `db:"user_id"`
	SentAt         time.Time `db:"sent_at"`
	SnippetCreated bool      `db:"snippet_created"`
}

/*
Stores the content of a Discord message for users with a linked
Discord account. Always created for users with a linked Discord
account, regardless of whether we create snippets or not.
*/
type DiscordMessageContent struct {
	MessageID   string `db:"message_id"`
	LastContent string `db:"last_content"`
	DiscordID   int    `db:"discord_id"`
}

type DiscordMessageAttachment struct {
	ID        string    `db:"id"`
	AssetID   uuid.UUID `db:"asset_id"`
	MessageID string    `db:"message_id"`
}

type DiscordMessageEmbed struct {
	ID          int        `db:"id"`
	Title       *string    `db:"title"`
	Description *string    `db:"description"`
	URL         *string    `db:"url"`
	ImageID     *uuid.UUID `db:"image_id"`
	MessageID   string     `db:"message_id"`
	VideoID     *uuid.UUID `db:"video_id"`
}

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

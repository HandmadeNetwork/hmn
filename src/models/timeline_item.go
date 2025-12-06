package models

import (
	"time"

	"github.com/google/uuid"
)

type TimelineItemType string

const (
	TimelineItemTypeSnippet TimelineItemType = "snippet"
	TimelineItemTypePost    TimelineItemType = "post"
	TimelineItemTypeStream  TimelineItemType = "stream" // NOTE(asaf): Not currently supported
)

// NOTE(asaf): This is a virtual model made up of several different tables
type TimelineItem struct {
	// Common
	// NOTE(asaf): Several different items can have the same ID because we're merging several tables
	ID                int              `db:"id"`
	Date              time.Time        `db:"\"when\""`
	Type              TimelineItemType `db:"timeline_type"`
	OwnerID           int              `db:"owner_id"`
	Title             string           `db:"title"`
	ParsedDescription string           `db:"parsed_desc"`
	RawDescription    string           `db:"raw_desc"`

	// Snippet
	AssetID          *uuid.UUID `db:"asset_id"`
	DiscordMessageID *string    `db:"discord_message_id"`
	ExternalUrl      *string    `db:"url"`

	// Post
	ProjectID  int        `db:"project_id"`
	ThreadID   int        `db:"thread_id"`
	SubforumID int        `db:"subforum_id"`
	ThreadType ThreadType `db:"thread_type"`
	FirstPost  bool       `db:"first_post"`
}

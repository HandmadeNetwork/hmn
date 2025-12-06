package models

import (
	"time"

	"github.com/google/uuid"
)

type SnippetProjectAssociationKind int

const (
	SnippetProjectKindDiscord SnippetProjectAssociationKind = iota + 1
	SnippetProjectKindWebsite
)

type Snippet struct {
	ID      int `db:"id"`
	OwnerID int `db:"owner_id"`

	When time.Time `db:"\"when\""`

	Description     string `db:"description"`
	DescriptionHtml string `db:"_description_html"`

	Url     *string    `db:"url"`
	AssetID *uuid.UUID `db:"asset_id"`

	EditedOnWebsite  bool    `db:"edited_on_website"`
	DiscordMessageID *string `db:"discord_message_id"`
}

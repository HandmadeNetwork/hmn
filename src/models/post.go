package models

import (
	"net"
	"time"
)

type Post struct {
	ID int `db:"id"`

	// TODO: Document each of these
	AuthorID   *int `db:"author_id"`
	CategoryID int  `db:"category_id"`
	ParentID   *int `db:"parent_id"`
	ThreadID   int  `db:"thread_id"`
	CurrentID  int  `db:"current_id"`
	ProjectID  int  `db:"project_id"`

	CategoryKind CategoryKind `db:"category_kind"`

	Depth        int       `db:"depth"`       // TODO: Drop this.
	Slug         string    `db:"slug"`        // TODO: Drop this.
	AuthorName   string    `db:"author_name"` // TODO: Drop this.
	PostDate     time.Time `db:"postdate"`    // TODO: Drop this.
	IP           net.IPNet `db:"ip"`          // TODO: Drop this.
	Sticky       bool      `db:"sticky"`      // TODO: Drop this.
	Deleted      bool      `db:"deleted"`
	Hits         int       `db:"hits"`         // TODO: Drop this.
	Featured     bool      `db:"featured"`     // TODO: Drop this.
	FeatureVotes int       `db:"featurevotes"` // TODO: Drop this.

	Preview  string `db:"preview"`
	ReadOnly bool   `db:"readonly"`
}

type PostVersion struct {
	ID     int `db:"id"`
	PostID int `db:"post_id"`

	TextRaw    string `db:"text_raw"`
	TextParsed string `db:"text_parsed"`

	EditIP     *net.IPNet `db:"edit_ip"`
	EditDate   time.Time  `db:"edit_date"`
	EditReason string     `db:"edit_reason"`
	EditorID   *int       `db:"editor_id"`
}

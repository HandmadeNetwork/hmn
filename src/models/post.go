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

	Depth        int       `db:"depth"`
	Slug         string    `db:"slug"`
	AuthorName   string    `db:"author_name"` // TODO: Drop this.
	PostDate     time.Time `db:"postdate"`
	IP           net.IPNet `db:"ip"`
	Sticky       bool      `db:"sticky"`
	Moderated    bool      `db:"moderated"` // TODO: I'm not sure this is ever meaningfully used. It always seems to be 0 / false?
	Hits         int       `db:"hits"`
	Featured     bool      `db:"featured"`
	FeatureVotes int       `db:"featurevotes"` // TODO: Remove this column from the db, it's never used

	Preview  string `db:"preview"`
	ReadOnly bool   `db:"readonly"`
}

type Parser int

const (
	ParserBBCode    Parser = 1
	ParserCleanHTML        = 2
	ParserMarkdown         = 3
)

type PostVersion struct {
	ID     int `db:"id"`
	PostID int `db:"post_id"`

	TextRaw    string `db:"text_raw"`
	TextParsed string `db:"text_parsed"`
	Parser     Parser `db:"parser"`

	EditIP     *net.IPNet `db:"edit_ip"`
	EditDate   time.Time  `db:"edit_date"`
	EditReason string     `db:"edit_reason"`
	EditorID   *int       `db:"editor_id"`
}

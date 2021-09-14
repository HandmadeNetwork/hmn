package models

import (
	"net"
	"time"
)

type Post struct {
	ID int `db:"id"`

	AuthorID  *int `db:"author_id"`
	ThreadID  int  `db:"thread_id"`
	CurrentID int  `db:"current_id"` // The id of the current PostVersion
	ProjectID int  `db:"project_id"`
	ReplyID   *int `db:"reply_id"`

	ThreadType ThreadType `db:"thread_type"`

	PostDate time.Time `db:"postdate"`
	Deleted  bool      `db:"deleted"`

	Preview  string `db:"preview"`
	ReadOnly bool   `db:"readonly"`
}

type PostVersion struct {
	ID     int `db:"id"`
	PostID int `db:"post_id"`

	TextRaw    string `db:"text_raw"`
	TextParsed string `db:"text_parsed"`

	IP         *net.IPNet `db:"ip"`
	Date       time.Time  `db:"date"`
	EditReason string     `db:"edit_reason"`
	EditorID   *int       `db:"editor_id"`
}

package models

type Thread struct {
	ID int `db:"id"`

	CategoryID int `db:"category_id"`

	Title      string `db:"title"`
	Hits       int    `db:"hits"`
	ReplyCount int    `db:"reply_count"`
	Sticky     bool   `db:"sticky"`
	Locked     bool   `db:"locked"`
	Moderated  int    `db:"moderated"`

	FirstID *int `db:"first_id"`
	LastID  *int `db:"last_id"`
}

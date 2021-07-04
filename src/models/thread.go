package models

type Thread struct {
	ID int `db:"id"`

	CategoryID int `db:"category_id"`

	Title   string `db:"title"`
	Sticky  bool   `db:"sticky"`
	Locked  bool   `db:"locked"`
	Deleted bool   `db:"deleted"`

	FirstID *int `db:"first_id"`
	LastID  *int `db:"last_id"`
}

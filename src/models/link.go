package models

type Link struct {
	ID        int    `db:"id"`
	Name      string `db:"name"`
	URL       string `db:"url"`
	Ordering  int    `db:"ordering"`
	Primary   bool   `db:"primary_link"`
	UserID    *int   `db:"user_id"`
	ProjectID *int   `db:"project_id"`
}

package models

type Link struct {
	ID        int     `db:"id"`
	Key       string  `db:"key"`
	Name      *string `db:"name"`
	Value     string  `db:"value"`
	Ordering  int     `db:"ordering"`
	UserID    *int    `db:"user_id"`
	ProjectID *int    `db:"project_id"`
}

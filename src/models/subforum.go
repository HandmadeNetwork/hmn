package models

type Subforum struct {
	ID int `db:"id"`

	ParentID  *int `db:"parent_id"`
	ProjectID int  `db:"project_id"`

	Slug  string `db:"slug"`
	Name  string `db:"name"`
	Blurb string `db:"blurb"`
}

package models

type LibraryResource struct {
	ID int `db:"id"`

	CategoryID int  `db:"category_id"`
	ProjectID  *int `db:"project_id"`

	Name          string `db:"name"`
	Description   string `db:"description"`
	Url           string `db:"url"`
	ContentType   string `db:"content_type"`
	Size          int    `db:"size"`
	IsDeleted     bool   `db:"is_deleted"`
	PreventsEmbed bool   `db:"prevents_embed"`
}

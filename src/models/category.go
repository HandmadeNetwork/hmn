package models

type CategoryType int

const (
	CatTypeBlog CategoryType = iota + 1
	CatTypeForum
	CatTypeStatic
	CatTypeAnnotation
	CatTypeWiki
	CatTypeLibraryResource
)

type Category struct {
	ID int `db:"id"`

	ParentID  *int `db:"parent_id"`
	ProjectID *int `db:"project_id"`

	Slug   *string      `db:"slug"`
	Name   *string      `db:"name"`
	Blurb  *string      `db:"blurb"`
	Kind   CategoryType `db:"kind"`
	Color1 string       `db:"color_1"`
	Color2 string       `db:"color_2"`
	Depth  int          `db:"depth"`
}

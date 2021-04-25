package models

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"github.com/jackc/pgx/v4/pgxpool"
)

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
	Depth  int          `db:"depth"` // TODO: What is this?
}

func (c *Category) GetParents(ctx context.Context, conn *pgxpool.Pool) []Category {
	type breadcrumbRow struct {
		Cat Category `db:"cats"`
	}
	rows, err := db.Query(ctx, conn, breadcrumbRow{},
		`
		WITH RECURSIVE cats AS (
				SELECT *
				FROM handmade_category AS cat
				WHERE cat.id = $1
			UNION ALL
				SELECT parentcat.*
				FROM
					handmade_category AS parentcat
					JOIN cats ON cats.parent_id = parentcat.id
		)
		SELECT $columns FROM cats;
		`,
		c.ID,
	)
	if err != nil {
		panic(err)
	}

	var result []Category
	for _, irow := range rows.ToSlice()[1:] {
		row := irow.(*breadcrumbRow)
		result = append(result, row.Cat)
	}

	return result
}

// func GetCategoryUrls(cats ...*Category) map[int]string {

// }

// func makeCategoryUrl(cat *Category, subdomain string) string {
// 	switch cat.Kind {
// 	case CatTypeBlog:
// 	case CatTypeForum:
// 	}
// 	return hmnurl.ProjectUrl("/flooger", nil, subdomain)
// }

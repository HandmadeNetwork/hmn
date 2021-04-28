package models

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"github.com/jackc/pgx/v4/pgxpool"
)

type CategoryKind int

const (
	CatKindBlog CategoryKind = iota + 1
	CatKindForum
	CatKindStatic
	CatKindAnnotation
	CatKindWiki
	CatKindLibraryResource
)

type Category struct {
	ID int `db:"id"`

	ParentID  *int `db:"parent_id"`
	ProjectID *int `db:"project_id"` // TODO: Make not null

	Slug   *string      `db:"slug"`  // TODO: Make not null
	Name   *string      `db:"name"`  // TODO: Make not null
	Blurb  *string      `db:"blurb"` // TODO: Make not null
	Kind   CategoryKind `db:"kind"`
	Color1 string       `db:"color_1"`
	Color2 string       `db:"color_2"`
	Depth  int          `db:"depth"` // TODO: What is this?
}

/*
Gets the category and its parent categories, starting from the root and working toward the
category itself. Useful for breadcrumbs and the like.
*/
func (c *Category) GetHierarchy(ctx context.Context, conn *pgxpool.Pool) []Category {
	// TODO: Make this work for a whole set of categories at once. Should be doable.
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

	rowsSlice := rows.ToSlice()
	var result []Category
	for i := len(rowsSlice) - 1; i >= 0; i-- {
		row := rowsSlice[i].(*breadcrumbRow)
		result = append(result, row.Cat)
	}

	return result
}

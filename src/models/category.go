package models

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
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
	ProjectID *int `db:"project_id"` // TODO: Make not null

	Slug   *string      `db:"slug"`  // TODO: Make not null
	Name   *string      `db:"name"`  // TODO: Make not null
	Blurb  *string      `db:"blurb"` // TODO: Make not null
	Kind   CategoryType `db:"kind"`
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

func GetCategoryUrls(ctx context.Context, conn *pgxpool.Pool, cats ...*Category) map[int]string {
	var projectIds []int
	for _, cat := range cats {
		id := *cat.ProjectID

		alreadyInList := false
		for _, otherId := range projectIds {
			if otherId == id {
				alreadyInList = true
				break
			}
		}

		if !alreadyInList {
			projectIds = append(projectIds, id)
		}
	}

	// TODO(inarray)!!!!!

	//for _, cat := range cats {
	//	hierarchy := makeCategoryUrl(cat.GetHierarchy(ctx, conn))
	//}

	return nil
}

func makeCategoryUrl(cats []*Category, subdomain string) string {
	path := ""
	for i, cat := range cats {
		if i == 0 {
			switch cat.Kind {
			case CatTypeBlog:
				path += "/blogs"
			case CatTypeForum:
				path += "/forums"
			// TODO: All cat types?
			default:
				return ""
			}
		} else {
			path += "/" + *cat.Slug
		}
	}

	return hmnurl.ProjectUrl(path, nil, subdomain)
}

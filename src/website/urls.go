package website

import (
	"context"
	"fmt"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/jackc/pgx/v4/pgxpool"
)

type categoryUrlQueryResult struct {
	Cat     models.Category `db:"cat"`
	Project models.Project  `db:"project"`
}

func GetAllCategoryUrls(ctx context.Context, conn *pgxpool.Pool) map[int]string {
	it, err := db.Query(ctx, conn, categoryUrlQueryResult{},
		`
		SELECT $columns
		FROM
			handmade_category AS cat
			JOIN handmade_project AS project ON project.id = cat.project_id
		`,
	)
	if err != nil {
		panic(err)
	}
	defer it.Close()

	return makeCategoryUrls(it.ToSlice())
}

func GetProjectCategoryUrls(ctx context.Context, conn *pgxpool.Pool, projectId ...int) map[int]string {
	it, err := db.Query(ctx, conn, categoryUrlQueryResult{},
		`
		SELECT $columns
		FROM
			handmade_category AS cat
			JOIN handmade_project AS project ON project.id = cat.project_id
		WHERE
			project.id = ANY ($1)
		`,
		projectId,
	)
	if err != nil {
		panic(err)
	}
	defer it.Close()

	return makeCategoryUrls(it.ToSlice())
}

func makeCategoryUrls(rows []interface{}) map[int]string {
	categories := make(map[int]*models.Category)
	for _, irow := range rows {
		cat := irow.(*categoryUrlQueryResult).Cat
		categories[cat.ID] = &cat
	}

	result := make(map[int]string)
	for _, irow := range rows {
		row := irow.(*categoryUrlQueryResult)

		// get hierarchy (backwards, so current -> parent -> root)
		var hierarchyReverse []*models.Category
		currentCatID := row.Cat.ID
		for {
			cat := categories[currentCatID]

			hierarchyReverse = append(hierarchyReverse, cat)
			if cat.ParentID == nil {
				break
			} else {
				currentCatID = *cat.ParentID
			}
		}

		// reverse to get root -> parent -> current
		hierarchy := make([]*models.Category, len(hierarchyReverse))
		for i := len(hierarchyReverse) - 1; i >= 0; i-- {
			hierarchy[len(hierarchyReverse)-1-i] = hierarchyReverse[i]
		}

		result[row.Cat.ID] = CategoryUrl(row.Project.Subdomain(), hierarchy...)
	}

	return result
}

func CategoryUrl(subdomain string, cats ...*models.Category) string {
	path := ""
	for i, cat := range cats {
		if i == 0 {
			switch cat.Kind {
			case models.CatKindBlog:
				path += "/blogs"
			case models.CatKindForum:
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

func PostUrl(post models.Post, catKind models.CategoryKind, categoryUrl string) string {
	categoryUrl = strings.TrimRight(categoryUrl, "/")

	switch catKind {
	// TODO: All the relevant post types. Maybe it doesn't make sense to lump them all together here.
	case models.CatKindBlog:
		return fmt.Sprintf("%s/p/%d/e/%d", categoryUrl, post.ThreadID, post.ID)
	case models.CatKindForum:
		return fmt.Sprintf("%s/t/%d/p/%d", categoryUrl, post.ThreadID, post.ID)
	}

	return ""
}
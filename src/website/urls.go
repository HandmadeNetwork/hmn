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
		WHERE
			cat.kind != 6
		`, // TODO(asaf): Clean up the db and remove the cat.kind != 6 check
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
			AND cat.kind != $2
		`, // TODO(asaf): Clean up the db and remove the cat.kind != library resource check
		projectId,
		models.CatKindLibraryResource,
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

		result[row.Cat.ID] = CategoryUrl(row.Project.Slug, hierarchy...)
	}

	return result
}

func CategoryUrl(projectSlug string, cats ...*models.Category) string {
	catSlugs := make([]string, 0, len(cats))
	for _, cat := range cats {
		catSlugs = append(catSlugs, *cat.Slug)
	}
	switch cats[0].Kind {
	case models.CatKindForum:
		return hmnurl.BuildForumCategory(projectSlug, catSlugs[1:], 1)
	default:
		return ""
	}
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

func ThreadUrl(thread models.Thread, catKind models.CategoryKind, categoryUrl string) string {
	categoryUrl = strings.TrimRight(categoryUrl, "/")

	switch catKind {
	// TODO: All the relevant post types. Maybe it doesn't make sense to lump them all together here.
	case models.CatKindBlog:
		return fmt.Sprintf("%s/p/%d", categoryUrl, thread.ID)
	case models.CatKindForum:
		return fmt.Sprintf("%s/t/%d", categoryUrl, thread.ID)
	}

	return ""
}

package models

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/oops"
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

type CategoryTree map[int]*CategoryTreeNode

type CategoryTreeNode struct {
	Category
	Parent *CategoryTreeNode
}

func (node *CategoryTreeNode) GetLineage() []*Category {
	current := node
	length := 0
	for current != nil {
		current = current.Parent
		length += 1
	}
	result := make([]*Category, length)
	current = node
	for i := length - 1; i >= 0; i -= 1 {
		result[i] = &current.Category
		current = current.Parent
	}
	return result
}

func GetFullCategoryTree(ctx context.Context, conn *pgxpool.Pool) CategoryTree {
	type categoryRow struct {
		Cat Category `db:"cat"`
	}
	rows, err := db.Query(ctx, conn, categoryRow{},
		`
		SELECT $columns
		FROM
			handmade_category as cat
		`,
	)
	if err != nil {
		panic(oops.New(err, "Failed to fetch category tree"))
	}

	rowsSlice := rows.ToSlice()
	catTreeMap := make(map[int]*CategoryTreeNode, len(rowsSlice))
	for _, row := range rowsSlice {
		cat := row.(*categoryRow).Cat
		catTreeMap[cat.ID] = &CategoryTreeNode{Category: cat}
	}

	for _, node := range catTreeMap {
		if node.ParentID != nil {
			node.Parent = catTreeMap[*node.ParentID]
		}
	}
	return catTreeMap
}

type CategoryLineageBuilder struct {
	Tree          CategoryTree
	CategoryCache map[int][]*Category
	SlugCache     map[int][]string
}

func MakeCategoryLineageBuilder(fullCategoryTree CategoryTree) *CategoryLineageBuilder {
	return &CategoryLineageBuilder{
		Tree:          fullCategoryTree,
		CategoryCache: make(map[int][]*Category),
		SlugCache:     make(map[int][]string),
	}
}

func (cl *CategoryLineageBuilder) GetLineage(catId int) []*Category {
	_, ok := cl.CategoryCache[catId]
	if !ok {
		cl.CategoryCache[catId] = cl.Tree[catId].GetLineage()
	}
	return cl.CategoryCache[catId]
}

func (cl *CategoryLineageBuilder) GetLineageSlugs(catId int) []string {
	_, ok := cl.SlugCache[catId]
	if !ok {
		lineage := cl.GetLineage(catId)
		result := make([]string, 0, len(lineage))
		for _, cat := range lineage {
			name := ""
			if cat.Slug != nil {
				name = *cat.Slug
			}
			result = append(result, name)
		}
		cl.SlugCache[catId] = result
	}
	return cl.SlugCache[catId]
}

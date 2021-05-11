package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
)

func BuildProjectMainCategoryUrl(projectSlug string, kind models.CategoryKind) string {
	switch kind {
	case models.CatKindBlog:
		return hmnurl.BuildBlog(projectSlug, 1)
	case models.CatKindForum:
		return hmnurl.BuildForumCategory(projectSlug, nil, 1)
	case models.CatKindWiki:
		return hmnurl.BuildWiki(projectSlug)
	case models.CatKindLibraryResource:
		return hmnurl.BuildLibrary(projectSlug)
	}
	return hmnurl.BuildProjectHomepage(projectSlug)
}

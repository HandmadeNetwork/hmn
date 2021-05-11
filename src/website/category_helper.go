package website

import (
	"git.handmade.network/hmn/hmn/src/models"
)

var CategoryKindDisplayNames = map[models.CategoryKind]string{
	models.CatKindBlog:            "Blog",
	models.CatKindForum:           "Forums",
	models.CatKindStatic:          "Static Page",
	models.CatKindAnnotation:      "Episode Guide",
	models.CatKindWiki:            "Wiki",
	models.CatKindLibraryResource: "Library",
}

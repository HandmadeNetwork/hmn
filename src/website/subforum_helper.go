package website

import (
	"git.handmade.network/hmn/hmn/src/models"
)

var ThreadTypeDisplayNames = map[models.ThreadType]string{
	models.ThreadTypeProjectArticle: "Blog",
	models.ThreadTypeForumPost:      "Forums",
}

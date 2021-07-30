package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
)

func BuildProjectRootResourceUrl(projectSlug string, kind models.ThreadType) string {
	switch kind {
	case models.ThreadTypeProjectArticle:
		return hmnurl.BuildBlog(projectSlug, 1)
	case models.ThreadTypeForumPost:
		return hmnurl.BuildForum(projectSlug, nil, 1)
	}
	return hmnurl.BuildProjectHomepage(projectSlug)
}

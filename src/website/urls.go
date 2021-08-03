package website

import (
	"math"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/utils"
)

func NumPages(numThings, thingsPerPage int) int {
	return utils.IntMax(int(math.Ceil(float64(numThings)/float64(thingsPerPage))), 1)
}

func BuildProjectRootResourceUrl(projectSlug string, kind models.ThreadType) string {
	switch kind {
	case models.ThreadTypeProjectBlogPost:
		return hmnurl.BuildBlog(projectSlug, 1)
	case models.ThreadTypeForumPost:
		return hmnurl.BuildForum(projectSlug, nil, 1)
	}
	return hmnurl.BuildProjectHomepage(projectSlug)
}

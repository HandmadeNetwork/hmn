package templates

import (
	"fmt"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
)

func PostUrl(post models.Post, catType models.CategoryType, subdomain string) string {
	switch catType {
	// TODO: All the relevant post types. Maybe it doesn't make sense to lump them all together here.
	case models.CatTypeBlog:
		return hmnurl.ProjectUrl(fmt.Sprintf("blogs/p/%d/e/%d", post.ThreadID, post.ID), nil, subdomain)
	case models.CatTypeForum:
		return hmnurl.ProjectUrl(fmt.Sprintf("forums/t/%d/p/%d", post.ThreadID, post.ID), nil, subdomain)
	}

	return ""
}

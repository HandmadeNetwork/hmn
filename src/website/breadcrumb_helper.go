package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

func ProjectBreadcrumb(projectUrlContext *hmnurl.UrlContext) templates.Breadcrumb {
	return templates.Breadcrumb{
		Name: projectUrlContext.ProjectName,
		Url:  projectUrlContext.BuildHomepage(),
	}
}

func ForumBreadcrumb(projectUrlContext *hmnurl.UrlContext) templates.Breadcrumb {
	return templates.Breadcrumb{
		Name: "Forums",
		Url:  projectUrlContext.BuildForum(nil, 1),
	}
}

func SubforumBreadcrumbs(projectUrlContext *hmnurl.UrlContext, lineageBuilder *models.SubforumLineageBuilder, subforumID int) []templates.Breadcrumb {
	var result []templates.Breadcrumb
	result = []templates.Breadcrumb{
		ProjectBreadcrumb(projectUrlContext),
		ForumBreadcrumb(projectUrlContext),
	}
	subforums := lineageBuilder.GetSubforumLineage(subforumID)
	slugs := lineageBuilder.GetSubforumLineageSlugs(subforumID)
	for i, subforum := range subforums {
		result = append(result, templates.Breadcrumb{
			Name: subforum.Name,
			Url:  projectUrlContext.BuildForum(slugs[0:i+1], 1),
		})
	}

	return result
}

func ForumThreadBreadcrumbs(projectUrlContext *hmnurl.UrlContext, lineageBuilder *models.SubforumLineageBuilder, thread *models.Thread) []templates.Breadcrumb {
	result := SubforumBreadcrumbs(projectUrlContext, lineageBuilder, *thread.SubforumID)
	result = append(result, templates.Breadcrumb{
		Name: thread.Title,
		Url:  projectUrlContext.BuildForumThread(lineageBuilder.GetSubforumLineageSlugs(*thread.SubforumID), thread.ID, thread.Title, 1),
	})
	return result
}

func BlogBreadcrumb(projectUrlContext *hmnurl.UrlContext) templates.Breadcrumb {
	return templates.Breadcrumb{
		Name: "Blog",
		Url:  projectUrlContext.BuildBlog(1),
	}
}

func BlogThreadBreadcrumbs(projectUrlContext *hmnurl.UrlContext, thread *models.Thread) []templates.Breadcrumb {
	result := []templates.Breadcrumb{
		BlogBreadcrumb(projectUrlContext),
		{Name: thread.Title, Url: projectUrlContext.BuildBlogThread(thread.ID, thread.Title)},
	}
	return result
}

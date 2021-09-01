package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

func ProjectBreadcrumb(project *models.Project) templates.Breadcrumb {
	return templates.Breadcrumb{
		Name: project.Name,
		Url:  hmnurl.BuildProjectHomepage(project.Slug),
	}
}

func ForumBreadcrumb(projectSlug string) templates.Breadcrumb {
	return templates.Breadcrumb{
		Name: "Forums",
		Url:  hmnurl.BuildForum(projectSlug, nil, 1),
	}
}

func SubforumBreadcrumbs(lineageBuilder *models.SubforumLineageBuilder, project *models.Project, subforumID int) []templates.Breadcrumb {
	var result []templates.Breadcrumb
	result = []templates.Breadcrumb{
		ProjectBreadcrumb(project),
		ForumBreadcrumb(project.Slug),
	}
	subforums := lineageBuilder.GetSubforumLineage(subforumID)
	slugs := lineageBuilder.GetSubforumLineageSlugs(subforumID)
	for i, subforum := range subforums {
		result = append(result, templates.Breadcrumb{
			Name: subforum.Name,
			Url:  hmnurl.BuildForum(project.Slug, slugs[0:i+1], 1),
		})
	}

	return result
}

func ForumThreadBreadcrumbs(lineageBuilder *models.SubforumLineageBuilder, project *models.Project, thread *models.Thread) []templates.Breadcrumb {
	result := SubforumBreadcrumbs(lineageBuilder, project, *thread.SubforumID)
	result = append(result, templates.Breadcrumb{
		Name: thread.Title,
		Url:  hmnurl.BuildForumThread(project.Slug, lineageBuilder.GetSubforumLineageSlugs(*thread.SubforumID), thread.ID, thread.Title, 1),
	})
	return result
}

func BlogBreadcrumb(projectSlug string) templates.Breadcrumb {
	return templates.Breadcrumb{
		Name: "Blog",
		Url:  hmnurl.BuildBlog(projectSlug, 1),
	}
}

func BlogThreadBreadcrumbs(projectSlug string, thread *models.Thread) []templates.Breadcrumb {
	result := []templates.Breadcrumb{
		BlogBreadcrumb(projectSlug),
		{Name: thread.Title, Url: hmnurl.BuildBlogThread(projectSlug, thread.ID, thread.Title)},
	}
	return result
}

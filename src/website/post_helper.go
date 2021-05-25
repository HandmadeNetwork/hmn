package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

// NOTE(asaf): Please don't use this if you already know the kind of the post beforehand. Just call the appropriate build function.
// You may pass 0 for `libraryResourceId` if the post is not a library resource post.
func UrlForGenericPost(post *models.Post, subforums []string, threadTitle string, libraryResourceId int, projectSlug string) string {
	switch post.CategoryKind {
	case models.CatKindBlog:
		return hmnurl.BuildBlogPost(projectSlug, post.ThreadID, post.ID)
	case models.CatKindForum:
		return hmnurl.BuildForumPost(projectSlug, subforums, post.ThreadID, post.ID)
	case models.CatKindWiki:
		if post.ParentID == nil {
			// NOTE(asaf): First post on a wiki "thread" is the wiki article itself
			return hmnurl.BuildWikiArticle(projectSlug, post.ThreadID, threadTitle)
		} else {
			// NOTE(asaf): Subsequent posts on a wiki "thread" are wiki talk posts
			return hmnurl.BuildWikiTalkPost(projectSlug, post.ThreadID, post.ID)
		}
	case models.CatKindLibraryResource:
		return hmnurl.BuildLibraryPost(projectSlug, libraryResourceId, post.ThreadID, post.ID)
	}

	return hmnurl.BuildProjectHomepage(projectSlug)
}

// NOTE(asaf): THIS DOESN'T HANDLE WIKI EDIT ITEMS. Wiki edits are PostTextVersions, not Posts.
func MakePostListItem(lineageBuilder *models.CategoryLineageBuilder, project *models.Project, thread *models.Thread, post *models.Post, user *models.User, libraryResource *models.LibraryResource, unread bool, includeBreadcrumbs bool, currentTheme string) templates.PostListItem {
	var result templates.PostListItem

	result.Title = thread.Title
	result.User = templates.UserToTemplate(user, currentTheme)
	result.Date = post.PostDate
	result.Unread = unread
	libraryResourceId := 0
	if libraryResource != nil {
		libraryResourceId = libraryResource.ID
	}
	result.Url = UrlForGenericPost(post, lineageBuilder.GetSubforumLineageSlugs(post.CategoryID), thread.Title, libraryResourceId, project.Slug)

	if includeBreadcrumbs {
		result.Breadcrumbs = append(result.Breadcrumbs, templates.Breadcrumb{
			Name: project.Name,
			Url:  hmnurl.BuildProjectHomepage(project.Slug),
		})
		result.Breadcrumbs = append(result.Breadcrumbs, templates.Breadcrumb{
			Name: CategoryKindDisplayNames[post.CategoryKind],
			Url:  BuildProjectMainCategoryUrl(project.Slug, post.CategoryKind),
		})
		switch post.CategoryKind {
		case models.CatKindForum:
			subforums := lineageBuilder.GetSubforumLineage(post.CategoryID)
			slugs := lineageBuilder.GetSubforumLineageSlugs(post.CategoryID)
			for i, subforum := range subforums {
				result.Breadcrumbs = append(result.Breadcrumbs, templates.Breadcrumb{
					Name: *subforum.Name, // NOTE(asaf): All subforum categories must have names.
					Url:  hmnurl.BuildForumCategory(project.Slug, slugs[0:i+1], 1),
				})
			}
		case models.CatKindLibraryResource:
			result.Breadcrumbs = append(result.Breadcrumbs, templates.Breadcrumb{
				Name: libraryResource.Name,
				Url:  hmnurl.BuildLibraryResource(project.Slug, libraryResource.ID),
			})
		}
	}

	return result
}

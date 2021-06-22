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

var PostTypeMap = map[models.CategoryKind][]templates.PostType{
	models.CatKindBlog:            []templates.PostType{templates.PostTypeBlogPost, templates.PostTypeBlogComment},
	models.CatKindForum:           []templates.PostType{templates.PostTypeForumThread, templates.PostTypeForumReply},
	models.CatKindWiki:            []templates.PostType{templates.PostTypeWikiCreate, templates.PostTypeWikiTalk},
	models.CatKindLibraryResource: []templates.PostType{templates.PostTypeLibraryComment, templates.PostTypeLibraryComment},
}

var PostTypePrefix = map[templates.PostType]string{
	templates.PostTypeBlogPost:       "New blog post",
	templates.PostTypeBlogComment:    "Blog comment",
	templates.PostTypeForumThread:    "New forum thread",
	templates.PostTypeForumReply:     "Forum reply",
	templates.PostTypeWikiCreate:     "New wiki page",
	templates.PostTypeWikiTalk:       "Wiki comment",
	templates.PostTypeWikiEdit:       "Wiki edit",
	templates.PostTypeLibraryComment: "Library comment",
}

func PostBreadcrumbs(lineageBuilder *models.CategoryLineageBuilder, project *models.Project, post *models.Post, libraryResource *models.LibraryResource) []templates.Breadcrumb {
	var result []templates.Breadcrumb
	result = append(result, templates.Breadcrumb{
		Name: project.Name,
		Url:  hmnurl.BuildProjectHomepage(project.Slug),
	})
	result = append(result, templates.Breadcrumb{
		Name: CategoryKindDisplayNames[post.CategoryKind],
		Url:  BuildProjectMainCategoryUrl(project.Slug, post.CategoryKind),
	})
	switch post.CategoryKind {
	case models.CatKindForum:
		subforums := lineageBuilder.GetSubforumLineage(post.CategoryID)
		slugs := lineageBuilder.GetSubforumLineageSlugs(post.CategoryID)
		for i, subforum := range subforums {
			result = append(result, templates.Breadcrumb{
				Name: *subforum.Name, // NOTE(asaf): All subforum categories must have names.
				Url:  hmnurl.BuildForumCategory(project.Slug, slugs[0:i+1], 1),
			})
		}
	case models.CatKindLibraryResource:
		result = append(result, templates.Breadcrumb{
			Name: libraryResource.Name,
			Url:  hmnurl.BuildLibraryResource(project.Slug, libraryResource.ID),
		})
	}
	return result
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
	result.Preview = post.Preview

	postType := templates.PostTypeUnknown
	postTypeOptions, found := PostTypeMap[post.CategoryKind]
	if found {
		var hasParent int
		if post.ParentID != nil {
			hasParent = 1
		}
		postType = postTypeOptions[hasParent]
	}
	result.PostType = postType
	result.PostTypePrefix = PostTypePrefix[result.PostType]

	if includeBreadcrumbs {
		result.Breadcrumbs = PostBreadcrumbs(lineageBuilder, project, post, libraryResource)
	}

	return result
}

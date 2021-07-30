package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

// NOTE(asaf): Please don't use this if you already know the kind of the post beforehand. Just call the appropriate build function.
func UrlForGenericPost(thread *models.Thread, post *models.Post, lineageBuilder *models.SubforumLineageBuilder, projectSlug string) string {
	switch post.ThreadType {
	case models.ThreadTypeProjectArticle:
		return hmnurl.BuildBlogPost(projectSlug, post.ThreadID, post.ID)
	case models.ThreadTypeForumPost:
		return hmnurl.BuildForumPost(projectSlug, lineageBuilder.GetSubforumLineageSlugs(*thread.SubforumID), post.ThreadID, post.ID)
	}

	return hmnurl.BuildProjectHomepage(projectSlug)
}

var PostTypeMap = map[models.ThreadType][]templates.PostType{
	//                               {         First post       ,       Subsequent post        }
	models.ThreadTypeProjectArticle: {templates.PostTypeBlogPost, templates.PostTypeBlogComment},
	models.ThreadTypeForumPost:      {templates.PostTypeForumThread, templates.PostTypeForumReply},
}

var PostTypePrefix = map[templates.PostType]string{
	templates.PostTypeBlogPost:    "New blog post",
	templates.PostTypeBlogComment: "Blog comment",
	templates.PostTypeForumThread: "New forum thread",
	templates.PostTypeForumReply:  "Forum reply",
}

func PostBreadcrumbs(lineageBuilder *models.SubforumLineageBuilder, project *models.Project, thread *models.Thread) []templates.Breadcrumb {
	var result []templates.Breadcrumb
	result = append(result, templates.Breadcrumb{
		Name: project.Name,
		Url:  hmnurl.BuildProjectHomepage(project.Slug),
	})
	result = append(result, templates.Breadcrumb{
		Name: ThreadTypeDisplayNames[thread.Type],
		Url:  BuildProjectRootResourceUrl(project.Slug, thread.Type),
	})
	switch thread.Type {
	case models.ThreadTypeForumPost:
		subforums := lineageBuilder.GetSubforumLineage(*thread.SubforumID)
		slugs := lineageBuilder.GetSubforumLineageSlugs(*thread.SubforumID)
		for i, subforum := range subforums {
			result = append(result, templates.Breadcrumb{
				Name: subforum.Name,
				Url:  hmnurl.BuildForum(project.Slug, slugs[0:i+1], 1),
			})
		}
	}
	return result
}

func MakePostListItem(
	lineageBuilder *models.SubforumLineageBuilder,
	project *models.Project,
	thread *models.Thread,
	post *models.Post,
	user *models.User,
	unread bool,
	includeBreadcrumbs bool,
	currentTheme string,
) templates.PostListItem {
	var result templates.PostListItem

	result.Title = thread.Title
	result.User = templates.UserToTemplate(user, currentTheme)
	result.Date = post.PostDate
	result.Unread = unread
	result.Url = UrlForGenericPost(thread, post, lineageBuilder, project.Slug)
	result.Preview = post.Preview

	postType := templates.PostTypeUnknown
	postTypeOptions, found := PostTypeMap[post.ThreadType]
	if found {
		isNotFirst := 0
		if *thread.FirstID != post.ID {
			isNotFirst = 1
		}
		postType = postTypeOptions[isNotFirst]
	}
	result.PostType = postType
	result.PostTypePrefix = PostTypePrefix[result.PostType]

	if includeBreadcrumbs {
		result.Breadcrumbs = PostBreadcrumbs(lineageBuilder, project, thread)
	}

	return result
}

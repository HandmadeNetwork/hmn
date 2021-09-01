package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

// NOTE(asaf): Please don't use this if you already know the kind of the post beforehand. Just call the appropriate build function.
func UrlForGenericPost(thread *models.Thread, post *models.Post, lineageBuilder *models.SubforumLineageBuilder, projectSlug string) string {
	switch post.ThreadType {
	case models.ThreadTypeProjectBlogPost:
		return hmnurl.BuildBlogThreadWithPostHash(projectSlug, post.ThreadID, thread.Title, post.ID)
	case models.ThreadTypeForumPost:
		return hmnurl.BuildForumPost(projectSlug, lineageBuilder.GetSubforumLineageSlugs(*thread.SubforumID), post.ThreadID, post.ID)
	}

	return hmnurl.BuildProjectHomepage(projectSlug)
}

var PostTypeMap = map[models.ThreadType][]templates.PostType{
	//                                {         First post       ,       Subsequent post        }
	models.ThreadTypeProjectBlogPost: {templates.PostTypeBlogPost, templates.PostTypeBlogComment},
	models.ThreadTypeForumPost:       {templates.PostTypeForumThread, templates.PostTypeForumReply},
}

var PostTypePrefix = map[templates.PostType]string{
	templates.PostTypeBlogPost:    "New blog post",
	templates.PostTypeBlogComment: "Blog comment",
	templates.PostTypeForumThread: "New forum thread",
	templates.PostTypeForumReply:  "Forum reply",
}

var ThreadTypeDisplayNames = map[models.ThreadType]string{
	models.ThreadTypeProjectBlogPost: "Blog",
	models.ThreadTypeForumPost:       "Forums",
}

func GenericThreadBreadcrumbs(lineageBuilder *models.SubforumLineageBuilder, project *models.Project, thread *models.Thread) []templates.Breadcrumb {
	var result []templates.Breadcrumb
	if thread.Type == models.ThreadTypeForumPost {
		result = SubforumBreadcrumbs(lineageBuilder, project, *thread.SubforumID)
	} else {
		result = []templates.Breadcrumb{
			{
				Name: project.Name,
				Url:  hmnurl.BuildProjectHomepage(project.Slug),
			},
			{
				Name: ThreadTypeDisplayNames[thread.Type],
				Url:  BuildProjectRootResourceUrl(project.Slug, thread.Type),
			},
		}
	}
	return result
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
		if thread.FirstID != post.ID {
			isNotFirst = 1
		}
		postType = postTypeOptions[isNotFirst]
	}
	result.PostType = postType
	result.PostTypePrefix = PostTypePrefix[result.PostType]

	if includeBreadcrumbs {
		result.Breadcrumbs = GenericThreadBreadcrumbs(lineageBuilder, project, thread)
	}

	return result
}

package website

import (
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

type PostInfo struct {
	ThreadType     models.ThreadType
	ThreadTitle    string
	ThreadID       int
	PostID         int
	SubforumID     int
	AuthorUsername string
	ProjectSlug    string
}

// NOTE(asaf): Please don't use these if you already know the kind of the thread beforehand. Just call the appropriate build function.
func UrlForGenericThread(urlContext *hmnurl.UrlContext, thread *models.Thread, lineageBuilder *models.SubforumLineageBuilder) string {
	switch thread.Type {
	case models.ThreadTypeProjectBlogPost:
		return urlContext.BuildBlogThread(thread.ID, thread.Title)
	case models.ThreadTypeForumPost:
		return urlContext.BuildForumThread(lineageBuilder.GetSubforumLineageSlugs(*thread.SubforumID), thread.ID, thread.Title, 1)
	}

	return urlContext.BuildHomepage()
}

func UrlForGenericPost(urlContext *hmnurl.UrlContext, thread *models.Thread, threadOwner *models.User, post *models.Post, lineageBuilder *models.SubforumLineageBuilder) string {
	switch post.ThreadType {
	case models.ThreadTypePersonalBlogPost:
		return hmnurl.BuildPersonalBlogThreadWithPostHash(threadOwner.Username, post.ThreadID, thread.Title, post.ID)
	case models.ThreadTypeProjectBlogPost:
		return urlContext.BuildBlogThreadWithPostHash(post.ThreadID, thread.Title, post.ID)
	case models.ThreadTypeForumPost:
		return urlContext.BuildForumPost(lineageBuilder.GetSubforumLineageSlugs(*thread.SubforumID), post.ThreadID, post.ID)
	}

	return urlContext.BuildHomepage()
}

var PostTypeMap = map[models.ThreadType][]templates.PostType{
	//                                {         First post       ,       Subsequent post        }
	models.ThreadTypeProjectBlogPost:  {templates.PostTypeBlogPost, templates.PostTypeBlogComment},
	models.ThreadTypePersonalBlogPost: {templates.PostTypePersonalBlogPost, templates.PostTypePersonalBlogComment},
	models.ThreadTypeForumPost:        {templates.PostTypeForumThread, templates.PostTypeForumReply},
}

var PostTypePrefix = map[templates.PostType]string{
	templates.PostTypeBlogPost:            "New blog post",
	templates.PostTypeBlogComment:         "Blog comment",
	templates.PostTypeForumThread:         "New forum thread",
	templates.PostTypeForumReply:          "Forum reply",
	templates.PostTypePersonalBlogPost:    "New personal blog post",
	templates.PostTypePersonalBlogComment: "Personal blog comment",
}

var ThreadTypeDisplayNames = map[models.ThreadType]string{
	models.ThreadTypeProjectBlogPost: "Blog",
	models.ThreadTypeForumPost:       "Forums",
}

func GenericThreadBreadcrumbs(urlContext *hmnurl.UrlContext, lineageBuilder *models.SubforumLineageBuilder, thread *models.Thread, threadOwner *models.User) []templates.Breadcrumb {
	var result []templates.Breadcrumb
	if thread.Type == models.ThreadTypeForumPost {
		result = SubforumBreadcrumbs(urlContext, lineageBuilder, *thread.SubforumID)
	} else if thread.Type == models.ThreadTypePersonalBlogPost {
		result = []templates.Breadcrumb{
			{
				Name: threadOwner.BestName(),
				Url:  hmnurl.BuildUserProfile(threadOwner.Username),
			},
			{
				Name: "Personal blog",
				Url:  hmnurl.BuildPersonalBlog(threadOwner.Username, 1),
			},
		}
	} else {
		result = []templates.Breadcrumb{
			{
				Name: urlContext.ProjectName,
				Url:  urlContext.BuildHomepage(),
			},
			{
				Name: ThreadTypeDisplayNames[thread.Type],
				Url:  BuildProjectRootResourceUrl(urlContext, thread.Type),
			},
		}
	}
	return result
}

func BuildProjectRootResourceUrl(urlContext *hmnurl.UrlContext, kind models.ThreadType) string {
	switch kind {
	case models.ThreadTypeProjectBlogPost:
		return urlContext.BuildBlog(1)
	case models.ThreadTypeForumPost:
		return urlContext.BuildForum(nil, 1)
	}
	return urlContext.BuildHomepage()
}

func MakePostListItem(
	lineageBuilder *models.SubforumLineageBuilder,
	project *models.Project,
	thread *models.Thread,
	threadOwner *models.User,
	post *models.Post,
	user *models.User,
	unread bool,
	includeBreadcrumbs bool,
) templates.PostListItem {
	var result templates.PostListItem

	urlContext := hmndata.UrlContextForProject(project)

	result.Title = thread.Title
	result.User = templates.UserToTemplate(user)
	result.Date = post.PostDate
	result.Unread = unread
	result.Url = UrlForGenericPost(urlContext, thread, threadOwner, post, lineageBuilder)
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
		result.Breadcrumbs = GenericThreadBreadcrumbs(urlContext, lineageBuilder, thread, threadOwner)
	}

	return result
}

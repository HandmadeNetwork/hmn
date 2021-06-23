package website

import (
	"html/template"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

var TimelineTypeMap = map[models.CategoryKind][]templates.TimelineType{
	//                                                                 No parent,                    Has parent
	models.CatKindBlog:            []templates.TimelineType{templates.TimelineTypeBlogPost, templates.TimelineTypeBlogComment},
	models.CatKindForum:           []templates.TimelineType{templates.TimelineTypeForumThread, templates.TimelineTypeForumReply},
	models.CatKindWiki:            []templates.TimelineType{templates.TimelineTypeWikiCreate, templates.TimelineTypeWikiTalk},
	models.CatKindLibraryResource: []templates.TimelineType{templates.TimelineTypeLibraryComment, templates.TimelineTypeLibraryComment},
}

var TimelineItemClassMap = map[templates.TimelineType]string{
	templates.TimelineTypeUnknown: "",

	templates.TimelineTypeForumThread: "forums",
	templates.TimelineTypeForumReply:  "forums",

	templates.TimelineTypeBlogPost:    "blogs",
	templates.TimelineTypeBlogComment: "blogs",

	templates.TimelineTypeWikiCreate: "wiki",
	templates.TimelineTypeWikiEdit:   "wiki",
	templates.TimelineTypeWikiTalk:   "wiki",

	templates.TimelineTypeLibraryComment: "library",

	templates.TimelineTypeSnippetImage:   "snippets",
	templates.TimelineTypeSnippetVideo:   "snippets",
	templates.TimelineTypeSnippetAudio:   "snippets",
	templates.TimelineTypeSnippetYoutube: "snippets",
}

var TimelineTypeTitleMap = map[templates.TimelineType]string{
	templates.TimelineTypeUnknown: "",

	templates.TimelineTypeForumThread: "New forums thread",
	templates.TimelineTypeForumReply:  "Forum reply",

	templates.TimelineTypeBlogPost:    "New blog post",
	templates.TimelineTypeBlogComment: "Blog comment",

	templates.TimelineTypeWikiCreate: "New wiki article",
	templates.TimelineTypeWikiEdit:   "Wiki edit",
	templates.TimelineTypeWikiTalk:   "Wiki talk",

	templates.TimelineTypeLibraryComment: "Library comment",

	templates.TimelineTypeSnippetImage:   "Snippet",
	templates.TimelineTypeSnippetVideo:   "Snippet",
	templates.TimelineTypeSnippetAudio:   "Snippet",
	templates.TimelineTypeSnippetYoutube: "Snippet",
}

func PostToTimelineItem(lineageBuilder *models.CategoryLineageBuilder, post *models.Post, thread *models.Thread, project *models.Project, libraryResource *models.LibraryResource, owner *models.User, currentTheme string) templates.TimelineItem {
	itemType := templates.TimelineTypeUnknown
	typeByCatKind, found := TimelineTypeMap[post.CategoryKind]
	if found {
		hasParent := 0
		if post.ParentID != nil {
			hasParent = 1
		}
		itemType = typeByCatKind[hasParent]
	}

	libraryResourceId := 0
	if libraryResource != nil {
		libraryResourceId = libraryResource.ID
	}

	return templates.TimelineItem{
		Type:      itemType,
		TypeTitle: TimelineTypeTitleMap[itemType],
		Class:     TimelineItemClassMap[itemType],
		Date:      post.PostDate,
		Url:       UrlForGenericPost(post, lineageBuilder.GetSubforumLineageSlugs(post.CategoryID), thread.Title, libraryResourceId, project.Slug),

		OwnerAvatarUrl: templates.UserAvatarUrl(owner, currentTheme),
		OwnerName:      templates.UserDisplayName(owner),
		OwnerUrl:       hmnurl.BuildUserProfile(owner.Username),
		Description:    "", // NOTE(asaf): No description for posts

		Title:       thread.Title,
		Breadcrumbs: PostBreadcrumbs(lineageBuilder, project, post, libraryResource),
	}
}

func PostVersionToWikiTimelineItem(lineageBuilder *models.CategoryLineageBuilder, version *models.PostVersion, post *models.Post, thread *models.Thread, project *models.Project, owner *models.User, currentTheme string) templates.TimelineItem {
	return templates.TimelineItem{
		Type:      templates.TimelineTypeWikiEdit,
		TypeTitle: TimelineTypeTitleMap[templates.TimelineTypeWikiEdit],
		Class:     TimelineItemClassMap[templates.TimelineTypeWikiEdit],
		Date:      version.EditDate,
		Url:       hmnurl.BuildWikiArticle(project.Slug, thread.ID, thread.Title),

		OwnerAvatarUrl: templates.UserAvatarUrl(owner, currentTheme),
		OwnerName:      templates.UserDisplayName(owner),
		OwnerUrl:       hmnurl.BuildUserProfile(owner.Username),
		Description:    "", // NOTE(asaf): No description for posts

		Title:       thread.Title,
		Breadcrumbs: PostBreadcrumbs(lineageBuilder, project, post, nil),
	}
}

var YoutubeRegex = regexp.MustCompile(`(?i)youtube\.com/watch\?.*v=(?P<videoid>[^/&]+)`)
var YoutubeShortRegex = regexp.MustCompile(`(?i)youtu\.be/(?P<videoid>[^/]+)`)

func SnippetToTimelineItem(snippet *models.Snippet, asset *models.Asset, discordMessage *models.DiscordMessage, owner *models.User, currentTheme string) templates.TimelineItem {
	itemType := templates.TimelineTypeUnknown
	youtubeId := ""
	assetUrl := ""
	mimeType := ""
	width := 0
	height := 0
	discordMessageUrl := ""

	if asset == nil {
		match := YoutubeRegex.FindStringSubmatch(*snippet.Url)
		index := YoutubeRegex.SubexpIndex("videoid")
		if match == nil {
			match = YoutubeShortRegex.FindStringSubmatch(*snippet.Url)
			index = YoutubeShortRegex.SubexpIndex("videoid")
		}
		if match != nil {
			youtubeId = match[index]
			itemType = templates.TimelineTypeSnippetYoutube
		}
	} else {
		if strings.HasPrefix(asset.MimeType, "image/") {
			itemType = templates.TimelineTypeSnippetImage
		} else if strings.HasPrefix(asset.MimeType, "video/") {
			itemType = templates.TimelineTypeSnippetVideo
		} else if strings.HasPrefix(asset.MimeType, "audio/") {
			itemType = templates.TimelineTypeSnippetAudio
		}
		assetUrl = hmnurl.BuildS3Asset(asset.S3Key)
		mimeType = asset.MimeType
		width = asset.Width
		height = asset.Height
	}

	if discordMessage != nil {
		discordMessageUrl = discordMessage.Url
	}

	return templates.TimelineItem{
		Type:  itemType,
		Class: TimelineItemClassMap[itemType],
		Date:  snippet.When,
		Url:   hmnurl.BuildSnippet(snippet.ID),

		OwnerAvatarUrl: templates.UserAvatarUrl(owner, currentTheme),
		OwnerName:      templates.UserDisplayName(owner),
		OwnerUrl:       hmnurl.BuildUserProfile(owner.Username),
		Description:    template.HTML(snippet.DescriptionHtml),

		DiscordMessageUrl: discordMessageUrl,
		Width:             width,
		Height:            height,
		AssetUrl:          assetUrl,
		MimeType:          mimeType,
		YoutubeID:         youtubeId,
	}
}

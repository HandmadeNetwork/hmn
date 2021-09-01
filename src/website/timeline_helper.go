package website

import (
	"html/template"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
)

var TimelineTypeMap = map[models.ThreadType][]templates.TimelineType{
	//                                {           First post         ,         Subsequent post          }
	models.ThreadTypeProjectBlogPost: {templates.TimelineTypeBlogPost, templates.TimelineTypeBlogComment},
	models.ThreadTypeForumPost:       {templates.TimelineTypeForumThread, templates.TimelineTypeForumReply},
}

var TimelineItemClassMap = map[templates.TimelineType]string{
	templates.TimelineTypeUnknown: "",

	templates.TimelineTypeForumThread: "forums",
	templates.TimelineTypeForumReply:  "forums",

	templates.TimelineTypeBlogPost:    "blogs",
	templates.TimelineTypeBlogComment: "blogs",

	templates.TimelineTypeSnippetImage:   "snippets",
	templates.TimelineTypeSnippetVideo:   "snippets",
	templates.TimelineTypeSnippetAudio:   "snippets",
	templates.TimelineTypeSnippetYoutube: "snippets",
}

var TimelineTypeTitleMap = map[templates.TimelineType]string{
	templates.TimelineTypeUnknown: "",

	templates.TimelineTypeForumThread: "New forum thread",
	templates.TimelineTypeForumReply:  "Forum reply",

	templates.TimelineTypeBlogPost:    "New blog post",
	templates.TimelineTypeBlogComment: "Blog comment",

	templates.TimelineTypeSnippetImage:   "Snippet",
	templates.TimelineTypeSnippetVideo:   "Snippet",
	templates.TimelineTypeSnippetAudio:   "Snippet",
	templates.TimelineTypeSnippetYoutube: "Snippet",
}

func PostToTimelineItem(lineageBuilder *models.SubforumLineageBuilder, post *models.Post, thread *models.Thread, project *models.Project, owner *models.User, currentTheme string) templates.TimelineItem {
	itemType := templates.TimelineTypeUnknown
	typeByCatKind, found := TimelineTypeMap[post.ThreadType]
	if found {
		isNotFirst := 0
		if thread.FirstID != post.ID {
			isNotFirst = 1
		}
		itemType = typeByCatKind[isNotFirst]
	}

	return templates.TimelineItem{
		Type:      itemType,
		TypeTitle: TimelineTypeTitleMap[itemType],
		Class:     TimelineItemClassMap[itemType],
		Date:      post.PostDate,
		Url:       UrlForGenericPost(thread, post, lineageBuilder, project.Slug),

		OwnerAvatarUrl: templates.UserAvatarUrl(owner, currentTheme),
		OwnerName:      owner.BestName(),
		OwnerUrl:       hmnurl.BuildUserProfile(owner.Username),
		Description:    "", // NOTE(asaf): No description for posts

		Title:       thread.Title,
		Breadcrumbs: GenericThreadBreadcrumbs(lineageBuilder, project, thread),
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
		OwnerName:      owner.BestName(),
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

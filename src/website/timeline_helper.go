package website

import (
	"context"
	"fmt"
	"html/template"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
)

func FetchFollowTimelineForUser(ctx context.Context, conn db.ConnOrTx, user *models.User, lineageBuilder *models.SubforumLineageBuilder) ([]templates.TimelineItem, error) {
	perf := perf.ExtractPerf(ctx)

	perf.StartBlock("FOLLOW", "Assemble follow data")
	following, err := db.Query[models.Follow](ctx, conn, `
		SELECT $columns
		FROM follower
		WHERE user_id = $1
	`, user.ID)

	if err != nil {
		return nil, oops.New(err, "failed to fetch follow data")
	}

	projectIDs := make([]int, 0, len(following))
	userIDs := make([]int, 0, len(following))
	for _, f := range following {
		if f.FollowingProjectID != nil {
			projectIDs = append(projectIDs, *f.FollowingProjectID)
		}
		if f.FollowingUserID != nil {
			userIDs = append(userIDs, *f.FollowingUserID)
		}
	}

	timelineItems := []templates.TimelineItem{}
	if len(userIDs)+len(projectIDs) > 0 {
		timelineItems, err = FetchTimeline(ctx, conn, user, lineageBuilder, hmndata.TimelineQuery{
			OwnerIDs:   userIDs,
			ProjectIDs: projectIDs,
		})
	}
	perf.EndBlock()

	return timelineItems, err
}

func FetchTimeline(ctx context.Context, conn db.ConnOrTx, currentUser *models.User, lineageBuilder *models.SubforumLineageBuilder, q hmndata.TimelineQuery) ([]templates.TimelineItem, error) {
	results, err := hmndata.FetchTimeline(ctx, conn, currentUser, q)
	if err != nil {
		logging.Error().Err(err).Msg("Fail")
	}
	if err != nil {
		return nil, err
	}

	timelineItems := make([]templates.TimelineItem, 0, len(results))
	for _, r := range results {
		timelineItems = append(timelineItems, TimelineItemToTemplate(r, lineageBuilder, false))
	}

	return timelineItems, nil
}

func FetchFollows(ctx context.Context, conn db.ConnOrTx, currentUser *models.User, userID int) ([]templates.Follow, error) {
	perf := perf.ExtractPerf(ctx)

	perf.StartBlock("SQL", "Fetch follows")
	following, err := db.Query[models.Follow](ctx, conn, `
		SELECT $columns
		FROM follower
		WHERE user_id = $1
	`, userID)
	if err != nil {
		return nil, oops.New(err, "failed to fetch follows")
	}
	perf.EndBlock()

	var userIDs, projectIDs []int
	for _, follow := range following {
		if follow.FollowingUserID != nil {
			userIDs = append(userIDs, *follow.FollowingUserID)
		}
		if follow.FollowingProjectID != nil {
			projectIDs = append(projectIDs, *follow.FollowingProjectID)
		}
	}

	var users []*models.User
	var projectsAndStuff []hmndata.ProjectAndStuff
	if len(userIDs) > 0 {
		users, err = hmndata.FetchUsers(ctx, conn, currentUser, hmndata.UsersQuery{
			UserIDs: userIDs,
		})
		if err != nil {
			return nil, oops.New(err, "failed to fetch users for follows")
		}
	}
	if len(projectIDs) > 0 {
		projectsAndStuff, err = hmndata.FetchProjects(ctx, conn, currentUser, hmndata.ProjectsQuery{
			ProjectIDs: projectIDs,
		})
		if err != nil {
			return nil, oops.New(err, "failed to fetch projects for follows")
		}
	}

	var result []templates.Follow
	for _, follow := range following {
		if follow.FollowingUserID != nil {
			for _, user := range users {
				if user.ID == *follow.FollowingUserID {
					u := templates.UserToTemplate(user)
					result = append(result, templates.Follow{
						User: &u,
					})
					break
				}
			}
		}
		if follow.FollowingProjectID != nil {
			for _, p := range projectsAndStuff {
				if p.Project.ID == *follow.FollowingProjectID {
					proj := templates.ProjectAndStuffToTemplate(&p)
					result = append(result, templates.Follow{
						Project: &proj,
					})
					break
				}
			}
		}
	}

	return result, nil
}

type TimelineTypeTitles struct {
	TypeTitleFirst    string
	TypeTitleNotFirst string
	FilterTitle       string
}

var TimelineTypeTitleMap = map[models.ThreadType]TimelineTypeTitles{
	models.ThreadTypeProjectBlogPost: {"New blog post", "Blog comment", "Blogs"},
	models.ThreadTypeForumPost:       {"New forum thread", "Forum reply", "Forums"},
}

func PostToTimelineItem(
	urlContext *hmnurl.UrlContext,
	lineageBuilder *models.SubforumLineageBuilder,
	post *models.Post,
	thread *models.Thread,
	owner *models.User,
) templates.TimelineItem {
	ownerTmpl := templates.UserToTemplate(owner)

	item := templates.TimelineItem{
		ID:          strconv.Itoa(post.ID),
		Date:        post.PostDate,
		Title:       thread.Title,
		Breadcrumbs: GenericThreadBreadcrumbs(urlContext, lineageBuilder, thread),
		Url:         UrlForGenericPost(urlContext, thread, post, lineageBuilder),

		OwnerAvatarUrl: ownerTmpl.AvatarUrl,
		OwnerName:      ownerTmpl.Name,
		OwnerUrl:       ownerTmpl.ProfileUrl,

		ForumLayout: true,
	}

	if typeTitles, ok := TimelineTypeTitleMap[post.ThreadType]; ok {
		if thread.FirstID == post.ID {
			item.TypeTitle = typeTitles.TypeTitleFirst
		} else {
			item.TypeTitle = typeTitles.TypeTitleNotFirst
		}
		item.FilterTitle = typeTitles.FilterTitle
	} else {
		logging.Warn().
			Int("postID", post.ID).
			Int("threadType", int(post.ThreadType)).
			Msg("unknown thread type for post")
	}

	return item
}

func TwitchStreamToTimelineItem(
	streamHistory *models.TwitchStreamHistory,
	ownerAvatarUrl string,
	ownerName string,
	ownerUrl string,
) templates.TimelineItem {
	url := fmt.Sprintf("https://twitch.tv/%s", streamHistory.TwitchLogin)
	title := fmt.Sprintf("%s is live on Twitch: %s", streamHistory.TwitchLogin, streamHistory.Title)
	desc := ""
	if streamHistory.StreamEnded {
		if streamHistory.VODUrl != "" {
			url = streamHistory.VODUrl
		}
		title = fmt.Sprintf("%s	was live on Twitch", streamHistory.TwitchLogin)

		streamDuration := streamHistory.EndedAt.Sub(streamHistory.StartedAt).Truncate(time.Second).String()
		desc = fmt.Sprintf("%s<br/><br/>Streamed for %s", streamHistory.Title, streamDuration)
	}
	item := templates.TimelineItem{
		ID:          streamHistory.StreamID,
		Date:        streamHistory.StartedAt,
		FilterTitle: "Live streams",
		Url:         url,
		Title:       title,
		Description: template.HTML(desc),

		OwnerAvatarUrl: ownerAvatarUrl,
		OwnerName:      ownerName,
		OwnerUrl:       ownerUrl,

		ForumLayout: true,
	}

	return item
}

func SnippetToTimelineItem(
	snippet *models.Snippet,
	asset *models.Asset,
	discordMessage *models.DiscordMessage,
	projects []*hmndata.ProjectAndStuff,
	owner *models.User,
	editable bool,
) templates.TimelineItem {
	item := templates.TimelineItem{
		ID:          strconv.Itoa(snippet.ID),
		Date:        snippet.When,
		FilterTitle: "Snippets",
		Url:         hmnurl.BuildSnippet(snippet.ID),

		OwnerAvatarUrl: templates.UserAvatarUrl(owner),
		OwnerName:      owner.BestName(),
		OwnerUrl:       hmnurl.BuildUserProfile(owner.Username),

		Description:    template.HTML(snippet.DescriptionHtml),
		RawDescription: snippet.Description,

		CanShowcase: true,
		Editable:    editable,
	}

	if asset != nil {
		if strings.HasPrefix(asset.MimeType, "image/") {
			item.Media = append(item.Media, imageMediaItem(asset))
		} else if strings.HasPrefix(asset.MimeType, "video/") {
			item.Media = append(item.Media, videoMediaItem(asset))
		} else if strings.HasPrefix(asset.MimeType, "audio/") {
			item.Media = append(item.Media, audioMediaItem(asset))
		} else {
			item.Media = append(item.Media, unknownMediaItem(asset))
		}
	}

	if snippet.Url != nil {
		url := *snippet.Url
		if videoId := getYoutubeVideoID(url); videoId != "" {
			item.Media = append(item.Media, youtubeMediaItem(videoId))
			item.CanShowcase = false
		}
	}

	if len(item.Media) == 0 ||
		(len(item.Media) > 0 && (item.Media[0].Width == 0 || item.Media[0].Height == 0)) {
		item.CanShowcase = false
	}

	if discordMessage != nil {
		item.DiscordMessageUrl = discordMessage.Url
	}

	sort.Slice(projects, func(i, j int) bool {
		return projects[i].Project.Name < projects[j].Project.Name
	})
	for _, proj := range projects {
		item.Projects = append(item.Projects, templates.ProjectAndStuffToTemplate(proj))
	}

	return item
}

var youtubeRegexes = [...]*regexp.Regexp{
	regexp.MustCompile(`(?i)youtube\.com/watch\?.*v=(?P<videoid>[^/&]+)`),
	regexp.MustCompile(`(?i)youtu\.be/(?P<videoid>[^/]+)`),
}

func getYoutubeVideoID(url string) string {
	for _, regex := range youtubeRegexes {
		match := regex.FindStringSubmatch(url)
		if match != nil {
			return match[regex.SubexpIndex("videoid")]
		}
	}

	return ""
}

func imageMediaItem(asset *models.Asset) templates.TimelineItemMedia {
	assetUrl := hmnurl.BuildS3Asset(asset.S3Key)

	return templates.TimelineItemMedia{
		Type:         templates.TimelineItemMediaTypeImage,
		AssetUrl:     assetUrl,
		ThumbnailUrl: assetUrl, // TODO: Use smaller thumbnails?
		MimeType:     asset.MimeType,
		Width:        asset.Width,
		Height:       asset.Height,
	}
}

func videoMediaItem(asset *models.Asset) templates.TimelineItemMedia {
	assetUrl := hmnurl.BuildS3Asset(asset.S3Key)
	var thumbnailUrl string
	if asset.ThumbnailS3Key != "" {
		thumbnailUrl = hmnurl.BuildS3Asset(asset.ThumbnailS3Key)
	}

	return templates.TimelineItemMedia{
		Type:         templates.TimelineItemMediaTypeVideo,
		AssetUrl:     assetUrl,
		ThumbnailUrl: thumbnailUrl,
		MimeType:     asset.MimeType,
		Width:        asset.Width,
		Height:       asset.Height,
	}
}

func audioMediaItem(asset *models.Asset) templates.TimelineItemMedia {
	assetUrl := hmnurl.BuildS3Asset(asset.S3Key)

	return templates.TimelineItemMedia{
		Type:     templates.TimelineItemMediaTypeAudio,
		AssetUrl: assetUrl,
		MimeType: asset.MimeType,
		Width:    asset.Width,
		Height:   asset.Height,
	}
}

func youtubeMediaItem(videoId string) templates.TimelineItemMedia {
	return templates.TimelineItemMedia{
		Type: templates.TimelineItemMediaTypeEmbed,
		EmbedHTML: template.HTML(fmt.Sprintf(
			`<iframe src="https://www.youtube-nocookie.com/embed/%s" allow="accelerometer; encrypted-media; gyroscope;" allowfullscreen frameborder="0"></iframe>`,
			template.HTMLEscapeString(videoId),
		)),
		ExtraOpenGraphItems: []templates.OpenGraphItem{
			{Property: "og:video", Value: fmt.Sprintf("https://youtube.com/watch?v=%s", videoId)},
			{Name: "twitter:card", Value: "player"},
		},
	}
}

func unknownMediaItem(asset *models.Asset) templates.TimelineItemMedia {
	assetUrl := hmnurl.BuildS3Asset(asset.S3Key)

	return templates.TimelineItemMedia{
		Type:     templates.TimelineItemMediaTypeUnknown,
		AssetUrl: assetUrl,
		MimeType: asset.MimeType,
		Filename: asset.Filename,
		FileSize: asset.Size,
	}
}

func TimelineItemToTemplate(item *hmndata.TimelineItemAndStuff, lineageBuilder *models.SubforumLineageBuilder, editable bool) templates.TimelineItem {
	filterTitle := ""
	typeTitle := ""
	url := ""
	var breadcrumbs []templates.Breadcrumb
	switch item.Item.Type {
	case models.TimelineItemTypeSnippet:
		filterTitle = "Snippets"
		typeTitle = "Snippet"
		url = hmnurl.BuildSnippet(item.Item.ID)

	case models.TimelineItemTypePost:
		urlContext := hmndata.UrlContextForProject(&item.Projects[0].Project)
		if item.Item.ThreadType == models.ThreadTypeProjectBlogPost {
			filterTitle = "Blogs"
			if item.Item.FirstPost {
				typeTitle = "New blog post"
			} else {
				typeTitle = "Blog comment"
			}
			url = urlContext.BuildBlogThreadWithPostHash(item.Item.ThreadID, item.Item.Title, item.Item.ID)
			breadcrumbs = []templates.Breadcrumb{
				{
					Name: urlContext.ProjectName,
					Url:  urlContext.BuildHomepage(),
				},
				{
					Name: "Blog",
					Url:  urlContext.BuildBlog(1),
				},
			}
		} else if item.Item.ThreadType == models.ThreadTypeForumPost {
			filterTitle = "Forums"
			if item.Item.FirstPost {
				typeTitle = "New forum thread"
			} else {
				typeTitle = "Forum reply"
			}
			url = urlContext.BuildForumPost(lineageBuilder.GetSubforumLineageSlugs(item.Item.SubforumID), item.Item.ThreadID, item.Item.ID)
			breadcrumbs = SubforumBreadcrumbs(urlContext, lineageBuilder, item.Item.SubforumID)
		}
	}

	ownerTmpl := templates.UserToTemplate(item.Owner)

	ti := templates.TimelineItem{
		ID:                strconv.Itoa(item.Item.ID),
		Date:              item.Item.Date,
		Title:             item.Item.Title,
		TypeTitle:         typeTitle,
		FilterTitle:       filterTitle,
		Breadcrumbs:       breadcrumbs,
		Url:               url,
		DiscordMessageUrl: "",

		OwnerAvatarUrl: ownerTmpl.AvatarUrl,
		OwnerName:      ownerTmpl.Name,
		OwnerUrl:       ownerTmpl.ProfileUrl,

		Projects:       nil,
		Description:    template.HTML(item.Item.ParsedDescription),
		RawDescription: item.Item.RawDescription,

		Media: nil,

		ForumLayout:         item.Item.Type == models.TimelineItemTypePost,
		AllowTitleWrap:      false,
		TruncateDescription: false,
		CanShowcase:         item.Item.Type == models.TimelineItemTypeSnippet,
		Editable:            item.Item.Type == models.TimelineItemTypeSnippet && editable,
	}

	if item.Asset != nil {
		if strings.HasPrefix(item.Asset.MimeType, "image/") {
			ti.Media = append(ti.Media, imageMediaItem(item.Asset))
		} else if strings.HasPrefix(item.Asset.MimeType, "video/") {
			ti.Media = append(ti.Media, videoMediaItem(item.Asset))
		} else if strings.HasPrefix(item.Asset.MimeType, "audio/") {
			ti.Media = append(ti.Media, audioMediaItem(item.Asset))
		} else {
			ti.Media = append(ti.Media, unknownMediaItem(item.Asset))
		}
	}

	if item.Item.ExternalUrl != nil {
		if videoId := getYoutubeVideoID(*item.Item.ExternalUrl); videoId != "" {
			ti.Media = append(ti.Media, youtubeMediaItem(videoId))
			ti.CanShowcase = false
		}
	}

	if len(ti.Media) == 0 ||
		(len(ti.Media) > 0 && (ti.Media[0].Width == 0 || ti.Media[0].Height == 0)) {
		ti.CanShowcase = false
	}

	if item.DiscordMessage != nil {
		ti.DiscordMessageUrl = item.DiscordMessage.Url
	}

	sort.Slice(item.Projects, func(i, j int) bool {
		return item.Projects[i].Project.Name < item.Projects[j].Project.Name
	})
	for _, proj := range item.Projects {
		ti.Projects = append(ti.Projects, templates.ProjectAndStuffToTemplate(proj))
	}

	return ti
}

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
	"git.handmade.network/hmn/hmn/src/utils"
)

func FetchFollowTimelineForUser(ctx context.Context, conn db.ConnOrTx, user *models.User) ([]templates.TimelineItem, error) {
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

	timelineItems, err := FetchTimeline(ctx, conn, user, TimelineQuery{
		UserIDs:    userIDs,
		ProjectIDs: projectIDs,
	})
	perf.EndBlock()

	return timelineItems, err
}

type TimelineQuery struct {
	UserIDs    []int
	ProjectIDs []int

	Limit int
}

func FetchTimeline(ctx context.Context, conn db.ConnOrTx, currentUser *models.User, q TimelineQuery) ([]templates.TimelineItem, error) {
	perf := perf.ExtractPerf(ctx)
	// var users []*models.User
	// var projects []hmndata.ProjectAndStuff
	var snippets []hmndata.SnippetAndStuff
	var posts []hmndata.PostAndStuff
	// var streamers []hmndata.TwitchStreamer
	// var streams []*models.TwitchStreamHistory
	var err error

	perf.StartBlock("TIMELINE", "Fetch timeline data")
	{
		snippets, err = hmndata.FetchSnippets(ctx, conn, currentUser, hmndata.SnippetQuery{
			OwnerIDs:   q.UserIDs,
			ProjectIDs: q.ProjectIDs,

			Limit: q.Limit,
		})
		if err != nil {
			return nil, oops.New(err, "failed to fetch timeline snippets")
		}

		posts, err = hmndata.FetchPosts(ctx, conn, currentUser, hmndata.PostsQuery{
			UserIDs:        q.UserIDs,
			ProjectIDs:     q.ProjectIDs,
			SortDescending: true,

			Limit: q.Limit,
		})
		if err != nil {
			return nil, oops.New(err, "failed to fetch timeline posts")
		}

		// streamers, err = hmndata.FetchTwitchStreamers(ctx, conn, hmndata.TwitchStreamersQuery{
		// 	UserIDs:    validUserIDs,
		// 	ProjectIDs: validProjectIDs,
		// })
		// if err != nil {
		// 	return nil, oops.New(err, "failed to fetch streamers")
		// }

		// twitchLogins := make([]string, 0, len(streamers))
		// for _, s := range streamers {
		// 	twitchLogins = append(twitchLogins, s.TwitchLogin)
		// }
		// streams, err = db.Query[models.TwitchStreamHistory](ctx, conn,
		// 	`
		// 	SELECT $columns FROM twitch_stream_history WHERE twitch_login = ANY ($1)
		// 	`,
		// 	twitchLogins,
		// )
		// if err != nil {
		// 	return nil, oops.New(err, "failed to fetch stream histories")
		// }
	}
	perf.EndBlock()

	perf.StartBlock("TIMELINE", "Construct timeline items")
	timelineItems := make([]templates.TimelineItem, 0, len(snippets)+len(posts))

	if len(posts) > 0 {
		perf.StartBlock("SQL", "Fetch subforum tree")
		subforumTree := models.GetFullSubforumTree(ctx, conn)
		lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
		perf.EndBlock()

		for _, post := range posts {
			timelineItems = append(timelineItems, PostToTimelineItem(
				hmndata.UrlContextForProject(&post.Project),
				lineageBuilder,
				&post.Post,
				&post.Thread,
				post.Author,
			))
		}
	}

	for _, s := range snippets {
		item := SnippetToTimelineItem(
			&s.Snippet,
			s.Asset,
			s.DiscordMessage,
			s.Projects,
			s.Owner,
			false,
		)
		timelineItems = append(timelineItems, item)
	}

	// for _, s := range streams {
	// 	ownerAvatarUrl := ""
	// 	ownerName := ""
	// 	ownerUrl := ""

	// 	for _, streamer := range streamers {
	// 		if streamer.TwitchLogin == s.TwitchLogin {
	// 			if streamer.UserID != nil {
	// 				for _, u := range users {
	// 					if u.ID == *streamer.UserID {
	// 						ownerAvatarUrl = templates.UserAvatarUrl(u)
	// 						ownerName = u.BestName()
	// 						ownerUrl = hmnurl.BuildUserProfile(u.Username)
	// 						break
	// 					}
	// 				}
	// 			} else if streamer.ProjectID != nil {
	// 				for _, p := range projects {
	// 					if p.Project.ID == *streamer.ProjectID {
	// 						ownerAvatarUrl = templates.ProjectLogoUrl(&p.Project, p.LogoLightAsset, p.LogoDarkAsset)
	// 						ownerName = p.Project.Name
	// 						ownerUrl = hmndata.UrlContextForProject(&p.Project).BuildHomepage()
	// 					}
	// 					break
	// 				}
	// 			}
	// 			break
	// 		}
	// 	}
	// 	item := TwitchStreamToTimelineItem(s, ownerAvatarUrl, ownerName, ownerUrl)
	// 	timelineItems = append(timelineItems, item)
	// }

	perf.StartBlock("TIMELINE", "Sort timeline")
	sort.Slice(timelineItems, func(i, j int) bool {
		return timelineItems[j].Date.Before(timelineItems[i].Date)
	})
	perf.EndBlock()
	perf.EndBlock()

	if q.Limit > 0 {
		timelineItems = utils.ClampSlice(timelineItems, q.Limit)
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

package website

import (
	"time"

	"git.handmade.network/hmn/hmn/src/templates"
)

func StyleTest(c *RequestContext) ResponseData {
	type tmpl struct {
		TestTimelineItems []templates.TimelineItem
	}

	var res ResponseData
	res.MustWriteTemplate("style_test.html", tmpl{
		TestTimelineItems: []templates.TimelineItem{
			// Forum post
			{
				OwnerName:      "Cool User",
				OwnerAvatarUrl: templates.UserAvatarDefaultUrl("dark"),
				Date:           time.Now().Add(-5 * time.Second),
				Breadcrumbs: []templates.Breadcrumb{
					{Name: "Project"},
					{Name: "Forums"},
					{Name: "Subforum"},
				},
				TypeTitle: "New forum post",
				Title:     "How can I a website?",
			},
			// Blog post
			// Snippet
			{
				SmallInfo:         true,
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Date(2022, 3, 20, 13, 32, 54, 0, time.UTC),
				Url:               "test",
				DiscordMessageUrl: "test",
				EmbedMedia: []templates.TimelineItemMedia{
					{
						Type:     templates.TimelineItemMediaTypeImage,
						AssetUrl: "https://assets.media.handmade.network/32ff3e7e-1d9c-4740-a062-1f8bec2e44cf/unknown.png",
					},
				},
			},
			// Snippet with embed
			{
				SmallInfo:         true,
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Date(2021, 4, 3, 1, 44, 54, 0, time.UTC),
				Url:               "test",
				DiscordMessageUrl: "test",
				EmbedMedia: []templates.TimelineItemMedia{
					youtubeMediaItem("FN9hZcTB16g"),
				},
				Projects: []templates.Project{
					{Name: "Cool Project", Logo: templates.UserAvatarDefaultUrl("light")},
				},
			},
			// Snippet with two images & multiple projects
			{
				SmallInfo:         true,
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Now().Add(-2 * 24 * time.Hour),
				Url:               "test",
				DiscordMessageUrl: "test",
				EmbedMedia: []templates.TimelineItemMedia{
					{
						Type:     templates.TimelineItemMediaTypeImage,
						AssetUrl: "https://assets.media.handmade.network/979d8850-f6b6-44b4-984e-93be82eb492b/PBR_WIP_20240620_01.png",
					},
					{
						Type:     templates.TimelineItemMediaTypeImage,
						AssetUrl: "https://assets.media.handmade.network/4cd4335d-c977-464b-994c-bda5a9b44b09/PBR_WIP_20240619_01.png",
					},
				},
				Projects: []templates.Project{
					{Name: "Cool Project", Logo: templates.UserAvatarDefaultUrl("light")},
					{Name: "Uncool Project"},
				},
			},
			// Snippet with a video and an image
			{
				SmallInfo:         true,
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Now().Add(-2 * time.Hour),
				Url:               "test",
				DiscordMessageUrl: "test",
				EmbedMedia: []templates.TimelineItemMedia{
					{
						Type:     templates.TimelineItemMediaTypeImage,
						AssetUrl: "https://assets.media.handmade.network/979d8850-f6b6-44b4-984e-93be82eb492b/PBR_WIP_20240620_01.png",
					},
					{
						Type:     templates.TimelineItemMediaTypeImage,
						AssetUrl: "https://assets.media.handmade.network/4cd4335d-c977-464b-994c-bda5a9b44b09/PBR_WIP_20240619_01.png",
					},
				},
				Projects: []templates.Project{
					{Name: "Cool Project", Logo: templates.UserAvatarDefaultUrl("light")},
					{Name: "Uncool Project"},
				},
			},
			// Snippet with every type of embed at once
		},
	}, c.Perf)
	return res
}

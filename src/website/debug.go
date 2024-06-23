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
				ForumLayout:    true,
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
			// Snippet with image
			{
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Date(2022, 3, 20, 13, 32, 54, 0, time.UTC),
				Url:               "test",
				DiscordMessageUrl: "test",
				Media: []templates.TimelineItemMedia{
					{
						Type:     templates.TimelineItemMediaTypeImage,
						AssetUrl: "https://assets.media.handmade.network/32ff3e7e-1d9c-4740-a062-1f8bec2e44cf/unknown.png",
					},
				},
				Projects: []templates.Project{
					{Name: "Cool Project", Logo: "https://assets.media.handmade.network/8c6a3b71-9e91-4bf6-80ef-bc8f3d21b30d/netsim.png"},
				},
			},
			// Snippet with tall image
			{
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Date(2022, 3, 20, 13, 32, 54, 0, time.UTC),
				Description:       "I got my LaGUI working on Android! 😄",
				Url:               "test",
				DiscordMessageUrl: "https://discord.com/channels/239737791225790464/404399251276169217/1245228715407966208",
				Media: []templates.TimelineItemMedia{
					{
						Type:     templates.TimelineItemMediaTypeImage,
						AssetUrl: "https://assets.media.handmade.network/ea6f914a-ea00-4cbb-bbd7-586b82fdb484/Screenshot_20240529_120344_com.lagui.simplest.jpg",
					},
				},
			},
			// Snippet with video
			{
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Date(2024, 1, 30, 3, 32, 54, 0, time.UTC),
				Url:               "test",
				Description:       "Using my newfound decoding knowledge I started working on a simple video editor. I also tried decoding 16 files at once, which didn't seem to bother my 3080 at all.",
				DiscordMessageUrl: "https://discord.com/channels/239737791225790464/404399251276169217/1249562779619168266",
				Media: []templates.TimelineItemMedia{
					{
						Type:         templates.TimelineItemMediaTypeVideo,
						AssetUrl:     "https://assets.media.handmade.network/b122c7be-dc6d-41fe-a5ed-033fe991927e/show16.mp4",
						ThumbnailUrl: "https://assets.media.handmade.network/b122c7be-dc6d-41fe-a5ed-033fe991927e/b122c7be-dc6d-41fe-a5ed-033fe991927e_thumb.jpg",
					},
				},
			},
			// Snippet with embed
			{
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Date(2021, 4, 3, 1, 44, 54, 0, time.UTC),
				Url:               "test",
				DiscordMessageUrl: "test",
				Media: []templates.TimelineItemMedia{
					youtubeMediaItem("FN9hZcTB16g"),
				},
			},
			// Snippet with two images & multiple projects
			{
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Now().Add(-2 * 24 * time.Hour),
				Url:               "test",
				DiscordMessageUrl: "test",
				Media: []templates.TimelineItemMedia{
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
					{Name: "Cool Project", Logo: "https://assets.media.handmade.network/8c6a3b71-9e91-4bf6-80ef-bc8f3d21b30d/netsim.png"},
					{Name: "Uncool Project"},
				},
			},
			// Snippet with a video and an image
			{
				OwnerName:         "Cool User",
				OwnerAvatarUrl:    templates.UserAvatarDefaultUrl("dark"),
				Date:              time.Now().Add(-2 * time.Hour),
				Url:               "test",
				DiscordMessageUrl: "test",
				Media: []templates.TimelineItemMedia{
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
					{Name: "Project without logo"},
				},
			},
			// Snippet with every type of embed at once
		},
	}, c.Perf)
	return res
}

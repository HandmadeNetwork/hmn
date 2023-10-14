package website

import (
	"fmt"
	"net/http"
	"net/url"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

func JamsIndex(c *RequestContext) ResponseData {
	var res ResponseData

	type TemplateData struct {
		templates.BaseData

		LispJamUrl string
		WRJ2021Url string
		WRJ2022Url string
		VJ2023Url  string
		WRJ2023Url string
	}

	res.MustWriteTemplate("jams_index.html", TemplateData{
		BaseData: getBaseDataAutocrumb(c, "Jams"),

		LispJamUrl: hmnurl.BuildFishbowl("lisp-jam"),
		WRJ2021Url: hmnurl.BuildJamIndex2021(),
		WRJ2022Url: hmnurl.BuildJamIndex2022(),
		VJ2023Url:  hmnurl.BuildJamIndex2023_Visibility(),
		WRJ2023Url: hmnurl.BuildJamIndex2023(),
	}, c.Perf)
	return res
}

func JamIndex2023(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilStart := daysUntil(hmndata.WRJ2023.StartTime)
	daysUntilEnd := daysUntil(hmndata.WRJ2023.EndTime)

	baseData := getBaseDataAutocrumb(c, hmndata.WRJ2023.Name)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2023/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam where we build software from scratch. September 25 - October 1 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2023()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int
		StartTimeUnix, EndTimeUnix   int64

		SubmittedProjectUrl  string
		ProjectSubmissionUrl string
		ShowcaseFeedUrl      string
		ShowcaseJson         string
		TwitchEmbedUrl       string

		JamProjects []templates.Project
	}

	var showcaseItems []templates.TimelineItem
	submittedProjectUrl := ""

	if c.CurrentUser != nil {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{hmndata.WRJ2023.Slug},
			Limit:    1,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
		}
		if len(projects) > 0 {
			urlContext := hmndata.UrlContextForProject(&projects[0].Project)
			submittedProjectUrl = urlContext.BuildHomepage()
		}
	}

	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.WRJ2023.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage(), c.Theme))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
			Limit:      12,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}
		showcaseItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, c.Theme, false)
			if timelineItem.CanShowcase {
				showcaseItems = append(showcaseItems, timelineItem)
			}
		}
	}

	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)

	twitchEmbedUrl := ""
	twitchStatus, err := db.QueryOne[models.TwitchLatestStatus](c, c.Conn,
		`
		SELECT $columns
		FROM twitch_latest_status
		WHERE twitch_login = $1
		`,
		"piratesoftware",
	)
	if err == nil {
		if twitchStatus.Live {
			hmnUrl, err := url.Parse(config.Config.BaseUrl)
			if err == nil {
				twitchEmbedUrl = fmt.Sprintf("https://player.twitch.tv/?channel=%s&parent=%s", twitchStatus.TwitchLogin, hmnUrl.Hostname())
			}
		}
	}

	res.MustWriteTemplate("jam_2023_wrj_index.html", JamPageData{
		BaseData:             baseData,
		DaysUntilStart:       daysUntilStart,
		DaysUntilEnd:         daysUntilEnd,
		StartTimeUnix:        hmndata.WRJ2023.StartTime.Unix(),
		EndTimeUnix:          hmndata.WRJ2023.EndTime.Unix(),
		ProjectSubmissionUrl: hmnurl.BuildProjectNewJam(),
		SubmittedProjectUrl:  submittedProjectUrl,
		ShowcaseFeedUrl:      hmnurl.BuildJamFeed2023(),
		ShowcaseJson:         showcaseJson,
		JamProjects:          pageProjects,
		TwitchEmbedUrl:       twitchEmbedUrl,
	}, c.Perf)
	return res
}

func JamFeed2023(c *RequestContext) ResponseData {
	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.WRJ2023.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	var timelineItems []templates.TimelineItem
	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}

		timelineItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, c.Theme, false)
			timelineItem.SmallInfo = true
			timelineItems = append(timelineItems, timelineItem)
		}
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage(), c.Theme))
	}

	type JamFeedData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int

		JamProjects   []templates.Project
		TimelineItems []templates.TimelineItem
	}

	daysUntilStart := daysUntil(hmndata.WRJ2023.StartTime)
	daysUntilEnd := daysUntil(hmndata.WRJ2023.EndTime)

	baseData := getBaseDataAutocrumb(c, hmndata.WRJ2023.Name)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2023/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. September 25 - October 1 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamFeed2023()},
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2023_wrj_feed.html", JamFeedData{
		BaseData:       baseData,
		DaysUntilStart: daysUntilStart,
		DaysUntilEnd:   daysUntilEnd,
		JamProjects:    pageProjects,
		TimelineItems:  timelineItems,
	}, c.Perf)
	return res
}

func JamIndex2023_Visibility(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilStart := daysUntil(hmndata.VJ2023.StartTime)
	daysUntilEnd := daysUntil(hmndata.VJ2023.EndTime)

	baseData := getBaseDataAutocrumb(c, hmndata.VJ2023.Name)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2023/opengraph.png", true)},
		{Property: "og:description", Value: "See things in a new way. April 14 - 16."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2023_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2023/TwitterCard.png", true)},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int
		StartTimeUnix, EndTimeUnix   int64

		SubmittedProjectUrl  string
		ProjectSubmissionUrl string
		ShowcaseFeedUrl      string
		ShowcaseJson         string
		RecapUrl             string

		JamProjects []templates.Project
	}

	var showcaseItems []templates.TimelineItem
	submittedProjectUrl := ""

	if c.CurrentUser != nil {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{hmndata.VJ2023.Slug},
			Limit:    1,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
		}
		if len(projects) > 0 {
			urlContext := hmndata.UrlContextForProject(&projects[0].Project)
			submittedProjectUrl = urlContext.BuildHomepage()
		}
	}

	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.VJ2023.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage(), c.Theme))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
			Limit:      12,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}
		showcaseItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, c.Theme, false)
			if timelineItem.CanShowcase {
				showcaseItems = append(showcaseItems, timelineItem)
			}
		}
	}

	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)

	res.MustWriteTemplate("jam_2023_vj_index.html", JamPageData{
		BaseData:             baseData,
		DaysUntilStart:       daysUntilStart,
		DaysUntilEnd:         daysUntilEnd,
		StartTimeUnix:        hmndata.VJ2023.StartTime.Unix(),
		EndTimeUnix:          hmndata.VJ2023.EndTime.Unix(),
		ProjectSubmissionUrl: hmnurl.BuildProjectNewJam(),
		SubmittedProjectUrl:  submittedProjectUrl,
		ShowcaseFeedUrl:      hmnurl.BuildJamFeed2023_Visibility(),
		ShowcaseJson:         showcaseJson,
		RecapUrl:             hmnurl.BuildJamRecap2023_Visibility(),
		JamProjects:          pageProjects,
	}, c.Perf)
	return res
}

func JamFeed2023_Visibility(c *RequestContext) ResponseData {
	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.VJ2023.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	var timelineItems []templates.TimelineItem
	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}

		timelineItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, c.Theme, false)
			timelineItem.SmallInfo = true
			timelineItems = append(timelineItems, timelineItem)
		}
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage(), c.Theme))
	}

	type JamFeedData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int

		JamUrl        string
		JamProjects   []templates.Project
		TimelineItems []templates.TimelineItem
	}

	daysUntilStart := daysUntil(hmndata.VJ2023.StartTime)
	daysUntilEnd := daysUntil(hmndata.VJ2023.EndTime)

	baseData := getBaseDataAutocrumb(c, hmndata.VJ2023.Name)

	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2023/opengraph.png", true)},
		{Property: "og:description", Value: "See things in a new way. April 14 - 16."},
		{Property: "og:url", Value: hmnurl.BuildJamFeed2023_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2023/TwitterCard.png", true)},
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2023_vj_feed.html", JamFeedData{
		BaseData:       baseData,
		DaysUntilStart: daysUntilStart,
		DaysUntilEnd:   daysUntilEnd,
		JamUrl:         hmnurl.BuildJamIndex2023_Visibility(),
		JamProjects:    pageProjects,
		TimelineItems:  timelineItems,
	}, c.Perf)
	return res
}

func JamRecap2023_Visibility(c *RequestContext) ResponseData {
	type JamRecapData struct {
		templates.BaseData

		JamUrl   string
		FeedUrl  string
		Ben      templates.User
		PostDate time.Time
	}

	var ben templates.User
	benUser, err := hmndata.FetchUserByUsername(c, c.Conn, c.CurrentUser, "bvisness", hmndata.UsersQuery{})
	if err == nil {
		ben = templates.UserToTemplate(benUser, c.Theme)
	} else if err == db.NotFound {
		ben = templates.UnknownUser
	} else {
		panic("where ben ???")
	}

	baseData := getBaseDataAutocrumb(c, hmndata.VJ2023.Name)

	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2023/opengraph.png", true)},
		// {Property: "og:description", Value: "See things in a new way. April 14 - 16."},
		{Property: "og:url", Value: hmnurl.BuildJamRecap2023_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2023/TwitterCard.png", true)},
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2023_vj_recap.html", JamRecapData{
		BaseData: baseData,
		JamUrl:   hmnurl.BuildJamIndex2023_Visibility(),
		FeedUrl:  hmnurl.BuildJamFeed2023_Visibility(),
		Ben:      ben,
		PostDate: time.Date(2023, 4, 22, 3, 9, 0, 0, time.UTC),
	}, c.Perf)
	return res
}

func JamIndex2022(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilStart := daysUntil(hmndata.WRJ2022.StartTime)
	daysUntilEnd := daysUntil(hmndata.WRJ2022.EndTime)

	baseData := getBaseDataAutocrumb(c, hmndata.WRJ2022.Name)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2022/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. August 15 - 21 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2022()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int
		StartTimeUnix, EndTimeUnix   int64

		SubmittedProjectUrl  string
		ProjectSubmissionUrl string
		ShowcaseFeedUrl      string
		ShowcaseJson         string

		JamProjects []templates.Project
	}

	var showcaseItems []templates.TimelineItem
	submittedProjectUrl := ""

	if c.CurrentUser != nil {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{hmndata.WRJ2022.Slug},
			Limit:    1,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
		}
		if len(projects) > 0 {
			urlContext := hmndata.UrlContextForProject(&projects[0].Project)
			submittedProjectUrl = urlContext.BuildHomepage()
		}
	}

	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.WRJ2022.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage(), c.Theme))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
			Limit:      12,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}
		showcaseItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, c.Theme, false)
			if timelineItem.CanShowcase {
				showcaseItems = append(showcaseItems, timelineItem)
			}
		}
	}

	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)

	res.MustWriteTemplate("jam_2022_wrj_index.html", JamPageData{
		BaseData:             baseData,
		DaysUntilStart:       daysUntilStart,
		DaysUntilEnd:         daysUntilEnd,
		StartTimeUnix:        hmndata.WRJ2022.StartTime.Unix(),
		EndTimeUnix:          hmndata.WRJ2022.EndTime.Unix(),
		ProjectSubmissionUrl: hmnurl.BuildProjectNewJam(),
		SubmittedProjectUrl:  submittedProjectUrl,
		ShowcaseFeedUrl:      hmnurl.BuildJamFeed2022(),
		ShowcaseJson:         showcaseJson,
		JamProjects:          pageProjects,
	}, c.Perf)
	return res
}

func JamFeed2022(c *RequestContext) ResponseData {
	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.WRJ2022.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	var timelineItems []templates.TimelineItem
	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}

		timelineItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, c.Theme, false)
			timelineItem.SmallInfo = true
			timelineItems = append(timelineItems, timelineItem)
		}
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage(), c.Theme))
	}

	type JamFeedData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int

		JamProjects   []templates.Project
		TimelineItems []templates.TimelineItem
	}

	daysUntilStart := daysUntil(hmndata.WRJ2022.StartTime)
	daysUntilEnd := daysUntil(hmndata.WRJ2022.EndTime)

	baseData := getBaseDataAutocrumb(c, hmndata.WRJ2022.Name)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2022/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. August 15 - 21 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamFeed2022()},
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2022_wrj_feed.html", JamFeedData{
		BaseData:       baseData,
		DaysUntilStart: daysUntilStart,
		DaysUntilEnd:   daysUntilEnd,
		JamProjects:    pageProjects,
		TimelineItems:  timelineItems,
	}, c.Perf)
	return res
}

func JamIndex2021(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilJam := daysUntil(hmndata.WRJ2021.StartTime)
	if daysUntilJam < 0 {
		daysUntilJam = 0
	}

	tagId := -1
	jamTag, err := hmndata.FetchTag(c, c.Conn, hmndata.TagQuery{
		Text: []string{"wheeljam"},
	})
	if err == nil {
		tagId = jamTag.ID
	} else {
		c.Logger.Warn().Err(err).Msg("failed to fetch jam tag; will fetch all snippets as a result")
	}

	snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
		Tags: []int{tagId},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam snippets"))
	}
	showcaseItems := make([]templates.TimelineItem, 0, len(snippets))
	for _, s := range snippets {
		timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, c.Theme, false)
		if timelineItem.CanShowcase {
			showcaseItems = append(showcaseItems, timelineItem)
		}
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SHOWCASE", "Convert to json")
	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)
	c.Perf.EndBlock()

	baseData := getBaseDataAutocrumb(c, hmndata.WRJ2021.Name)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2021/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to bring a fresh perspective to old ideas. September 27 - October 3 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2021()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntil         int
		ShowcaseItemsJSON string
	}

	res.MustWriteTemplate("jam_2021_wrj_index.html", JamPageData{
		BaseData:          baseData,
		DaysUntil:         daysUntilJam,
		ShowcaseItemsJSON: showcaseJson,
	}, c.Perf)
	return res
}

func daysUntil(t time.Time) int {
	d := t.Sub(time.Now())
	if d < 0 {
		d = 0
	}
	return int(utils.DurationRoundUp(d, 24*time.Hour) / (24 * time.Hour))
}

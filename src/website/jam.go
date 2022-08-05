package website

import (
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

func JamIndex2022(c *RequestContext) ResponseData {
	var res ResponseData

	// If logged in, fetch jam project
	// Link to project page if found, otherwise link to project creation page with ?jam=1

	daysUntilStart := daysUntil(hmndata.WRJ2022.StartTime)
	daysUntilEnd := daysUntil(hmndata.WRJ2022.EndTime)

	baseData := getBaseDataAutocrumb(c, hmndata.WRJ2022.Name)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade.Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2022/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. August 15 - 21 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int
		StartTimeUnix, EndTimeUnix   int64
		SubmittedProjectUrl          string
		ProjectSubmissionUrl         string
		ShowcaseFeedUrl              string
		ShowcaseJson                 string
	}

	var showcaseItems []templates.TimelineItem
	submittedProjectUrl := ""
	if daysUntilStart <= 0 && daysUntilEnd > 0 {
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

		jamProjectTags := make([]int, 0, len(jamProjects))
		for _, jp := range jamProjects {
			if jp.Tag != nil {
				jamProjectTags = append(jamProjectTags, jp.Tag.ID)
			}
		}

		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			Tags:  jamProjectTags,
			Limit: 12,
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

	res.MustWriteTemplate("wheeljam_2022_index.html", JamPageData{
		BaseData:             baseData,
		DaysUntilStart:       daysUntilStart,
		DaysUntilEnd:         daysUntilEnd,
		StartTimeUnix:        hmndata.WRJ2022.StartTime.Unix(),
		EndTimeUnix:          hmndata.WRJ2022.EndTime.Unix(),
		ProjectSubmissionUrl: hmnurl.BuildProjectNewJam(),
		SubmittedProjectUrl:  submittedProjectUrl,
		ShowcaseFeedUrl:      hmnurl.BuildJamFeed2022(),
		ShowcaseJson:         showcaseJson,
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

	snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
		ProjectIDs: projectIds,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
	}

	timelineItems := make([]templates.TimelineItem, 0, len(snippets))

	for _, s := range snippets {
		timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, c.Theme, false)
		timelineItem.SmallInfo = true
		timelineItems = append(timelineItems, timelineItem)
	}

	// TODO(asaf): add forum posts from jam project threads to timeline
	// TODO(asaf): Sort timeline items

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
		{Property: "og:site_name", Value: "Handmade.Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2022/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. August 15 - 21 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex()},
	}

	var res ResponseData
	res.MustWriteTemplate("wheeljam_2022_feed.html", JamFeedData{
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
		{Property: "og:site_name", Value: "Handmade.Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2021/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to bring a fresh perspective to old ideas. September 27 - October 3 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntil         int
		ShowcaseItemsJSON string
	}

	res.MustWriteTemplate("wheeljam_2021_index.html", JamPageData{
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

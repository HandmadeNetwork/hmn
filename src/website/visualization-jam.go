package website

import (
	"net/http"
	//	"time"

	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	// "git.handmade.network/hmn/hmn/src/utils"
)

func VisualizationIndex2023(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilStart := daysUntil(hmndata.VJ2023.StartTime)
	daysUntilEnd := daysUntil(hmndata.VJ2023.EndTime)

	baseData := getBaseDataAutocrumb(c, hmndata.VJ2023.Name)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade.Network"},
		{Property: "og:type", Value: "website"},
		// TODO:
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2022/opengraph.png", true)},
		{Property: "og:description", Value: "See things in a new way. April 14 - 16."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex()},
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

	res.MustWriteTemplate("visualization_jam_2023.html", JamPageData{
		BaseData:             baseData,
		DaysUntilStart:       daysUntilStart,
		DaysUntilEnd:         daysUntilEnd,
		StartTimeUnix:        hmndata.VJ2023.StartTime.Unix(),
		EndTimeUnix:          hmndata.VJ2023.EndTime.Unix(),
		ProjectSubmissionUrl: hmnurl.BuildProjectNewJam(),
		SubmittedProjectUrl:  submittedProjectUrl,
		ShowcaseFeedUrl:      hmnurl.BuildJamFeed2022(),
		ShowcaseJson:         showcaseJson,
		JamProjects:          pageProjects,
	}, c.Perf)
	return res
}

// func daysUntil(t time.Time) int {
// 	d := t.Sub(time.Now())
// 	if d < 0 {
// 		d = 0
// 	}
// 	return int(utils.DurationRoundUp(d, 24*time.Hour) / (24 * time.Hour))
// }

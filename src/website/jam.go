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

	jamStartTime := time.Date(2022, 8, 15, 0, 0, 0, 0, time.UTC)
	jamEndTime := time.Date(2022, 8, 22, 0, 0, 0, 0, time.UTC)
	daysUntilStart := daysUntil(jamStartTime)
	daysUntilEnd := daysUntil(jamEndTime)

	baseData := getBaseDataAutocrumb(c, "Wheel Reinvention Jam 2022")
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2022/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. August 15 - 21 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int
	}

	res.MustWriteTemplate("wheeljam_2022_index.html", JamPageData{
		BaseData:       baseData,
		DaysUntilStart: daysUntilStart,
		DaysUntilEnd:   daysUntilEnd,
	}, c.Perf)
	return res
}

func JamIndex2021(c *RequestContext) ResponseData {
	var res ResponseData

	jamStartTime := time.Date(2021, 9, 27, 0, 0, 0, 0, time.UTC)
	daysUntilJam := daysUntil(jamStartTime)
	if daysUntilJam < 0 {
		daysUntilJam = 0
	}

	tagId := -1
	jamTag, err := hmndata.FetchTag(c.Context(), c.Conn, hmndata.TagQuery{
		Text: []string{"wheeljam"},
	})
	if err == nil {
		tagId = jamTag.ID
	} else {
		c.Logger.Warn().Err(err).Msg("failed to fetch jam tag; will fetch all snippets as a result")
	}

	snippets, err := hmndata.FetchSnippets(c.Context(), c.Conn, c.CurrentUser, hmndata.SnippetQuery{
		Tags: []int{tagId},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam snippets"))
	}
	showcaseItems := make([]templates.TimelineItem, 0, len(snippets))
	for _, s := range snippets {
		timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Tags, s.Owner, c.Theme)
		if timelineItem.CanShowcase {
			showcaseItems = append(showcaseItems, timelineItem)
		}
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SHOWCASE", "Convert to json")
	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)
	c.Perf.EndBlock()

	baseData := getBaseDataAutocrumb(c, "Wheel Reinvention Jam")
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

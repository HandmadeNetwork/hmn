package website

import (
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func JamIndex(c *RequestContext) ResponseData {
	var res ResponseData

	jamStartTime := time.Date(2021, 9, 27, 0, 0, 0, 0, time.UTC)
	daysUntil := jamStartTime.YearDay() - time.Now().UTC().YearDay()
	if daysUntil < 0 {
		daysUntil = 0
	}

	var tagIds []int
	jamTag, err := FetchTag(c.Context(), c.Conn, "wheeljam")
	if err == nil {
		tagIds = []int{jamTag.ID}
	} else {
		c.Logger.Warn().Err(err).Msg("failed to fetch jam tag; will fetch all snippets as a result")
	}

	snippets, err := FetchSnippets(c.Context(), c.Conn, c.CurrentUser, SnippetQuery{
		Tags: tagIds,
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
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to bring a fresh perspective to old ideas. September 27 - October 3 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntil         int
		ShowcaseItemsJSON string
	}

	res.MustWriteTemplate("wheeljam_index.html", JamPageData{
		BaseData:          baseData,
		DaysUntil:         daysUntil,
		ShowcaseItemsJSON: showcaseJson,
	}, c.Perf)
	return res
}

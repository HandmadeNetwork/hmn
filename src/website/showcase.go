package website

import (
	"net/http"

	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

type ShowcaseData struct {
	templates.BaseData
	ShowcaseItems       string // NOTE(asaf): JSON string
	ShowcaseAtomFeedUrl string
}

func Showcase(c *RequestContext) ResponseData {
	snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets"))
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
	jsonItems := templates.TimelineItemsToJSON(showcaseItems)
	c.Perf.EndBlock()

	baseData := getBaseDataAutocrumb(c, "Community Showcase")
	var res ResponseData
	res.MustWriteTemplate("showcase.html", ShowcaseData{
		BaseData:            baseData,
		ShowcaseItems:       jsonItems,
		ShowcaseAtomFeedUrl: hmnurl.BuildAtomFeedForShowcase(),
	}, c.Perf)
	return res
}

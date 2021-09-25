package website

import (
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
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

	c.Perf.StartBlock("SQL", "Fetch showcase snippets")
	type snippetQuery struct {
		Owner          models.User            `db:"owner"`
		Snippet        models.Snippet         `db:"snippet"`
		Asset          *models.Asset          `db:"asset"`
		DiscordMessage *models.DiscordMessage `db:"discord_message"`
	}
	snippetQueryResult, err := db.Query(c.Context(), c.Conn, snippetQuery{},
		`
		SELECT $columns
		FROM
			handmade_snippet AS snippet
			INNER JOIN auth_user AS owner ON owner.id = snippet.owner_id
			LEFT JOIN handmade_asset AS asset ON asset.id = snippet.asset_id
			LEFT JOIN handmade_discordmessage AS discord_message ON discord_message.id = snippet.discord_message_id
		WHERE
			snippet.is_jam
		ORDER BY snippet.when DESC
		LIMIT 20
		`,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam snippets"))
	}
	snippetQuerySlice := snippetQueryResult.ToSlice()
	showcaseItems := make([]templates.TimelineItem, 0, len(snippetQuerySlice))
	for _, s := range snippetQuerySlice {
		row := s.(*snippetQuery)
		timelineItem := SnippetToTimelineItem(&row.Snippet, row.Asset, row.DiscordMessage, &row.Owner, c.Theme)
		if timelineItem.Type != templates.TimelineTypeSnippetYoutube {
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

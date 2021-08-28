package website

import (
	"net/http"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

type ShowcaseData struct {
	templates.BaseData
	ShowcaseItems       string // NOTE(asaf): JSON string
	ShowcaseAtomFeedUrl string
}

func Showcase(c *RequestContext) ResponseData {
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
		ORDER BY snippet.when DESC
		`,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets"))
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
	jsonItems := templates.TimelineItemsToJSON(showcaseItems)
	c.Perf.EndBlock()

	baseData := getBaseData(c)
	baseData.Title = "Community Showcase"
	var res ResponseData
	res.MustWriteTemplate("showcase.html", ShowcaseData{
		BaseData:            baseData,
		ShowcaseItems:       jsonItems,
		ShowcaseAtomFeedUrl: hmnurl.BuildAtomFeedForShowcase(),
	}, c.Perf)
	return res
}

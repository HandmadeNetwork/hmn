package website

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

type SnippetData struct {
	templates.BaseData
	Snippet templates.TimelineItem
}

func Snippet(c *RequestContext) ResponseData {
	snippetId := -1
	snippetIdStr, found := c.PathParams["snippetid"]
	if found && snippetIdStr != "" {
		var err error
		if snippetId, err = strconv.Atoi(snippetIdStr); err != nil {
			return FourOhFour(c)
		}
	}
	if snippetId < 1 {
		return FourOhFour(c)
	}

	c.Perf.StartBlock("SQL", "Fetch snippet")
	type snippetQuery struct {
		Owner          models.User            `db:"owner"`
		Snippet        models.Snippet         `db:"snippet"`
		Asset          *models.Asset          `db:"asset"`
		DiscordMessage *models.DiscordMessage `db:"discord_message"`
	}
	snippetQueryResult, err := db.QueryOne(c.Context(), c.Conn, snippetQuery{},
		`
		SELECT $columns
		FROM
			handmade_snippet AS snippet
			INNER JOIN auth_user AS owner ON owner.id = snippet.owner_id
			LEFT JOIN handmade_asset AS asset ON asset.id = snippet.asset_id
			LEFT JOIN handmade_discordmessage AS discord_message ON discord_message.id = snippet.discord_message_id
		WHERE snippet.id = $1
		`,
		snippetId,
	)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return FourOhFour(c)
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippet"))
		}
	}
	c.Perf.EndBlock()

	snippetData := snippetQueryResult.(*snippetQuery)

	snippet := SnippetToTimelineItem(&snippetData.Snippet, snippetData.Asset, snippetData.DiscordMessage, &snippetData.Owner, c.Theme)

	opengraph := []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade.Network"},
		{Property: "og:type", Value: "article"},
		{Property: "og:url", Value: snippet.Url},
		{Property: "og:title", Value: fmt.Sprintf("Snippet by %s", snippet.OwnerName)},
		{Property: "og:description", Value: string(snippet.Description)},
	}

	if snippet.Type == templates.TimelineTypeSnippetImage {
		opengraphImage := []templates.OpenGraphItem{
			{Property: "og:image", Value: snippet.AssetUrl},
			{Property: "og:image:width", Value: strconv.Itoa(snippet.Width)},
			{Property: "og:image:height", Value: strconv.Itoa(snippet.Height)},
			{Property: "og:image:type", Value: snippet.MimeType},
			{Name: "twitter:card", Value: "summary_large_image"},
		}
		opengraph = append(opengraph, opengraphImage...)
	} else if snippet.Type == templates.TimelineTypeSnippetVideo {
		opengraphVideo := []templates.OpenGraphItem{
			{Property: "og:video", Value: snippet.AssetUrl},
			{Property: "og:video:width", Value: strconv.Itoa(snippet.Width)},
			{Property: "og:video:height", Value: strconv.Itoa(snippet.Height)},
			{Property: "og:video:type", Value: snippet.MimeType},
			{Name: "twitter:card", Value: "player"},
		}
		opengraph = append(opengraph, opengraphVideo...)
	} else if snippet.Type == templates.TimelineTypeSnippetAudio {
		opengraphAudio := []templates.OpenGraphItem{
			{Property: "og:audio", Value: snippet.AssetUrl},
			{Property: "og:audio:type", Value: snippet.MimeType},
			{Name: "twitter:card", Value: "player"},
		}
		opengraph = append(opengraph, opengraphAudio...)
	} else if snippet.Type == templates.TimelineTypeSnippetYoutube {
		opengraphYoutube := []templates.OpenGraphItem{
			{Property: "og:video", Value: fmt.Sprintf("https://youtube.com/watch?v=%s", snippet.YoutubeID)},
			{Name: "twitter:card", Value: "player"},
		}
		opengraph = append(opengraph, opengraphYoutube...)
	}

	baseData := getBaseData(
		c,
		fmt.Sprintf("Snippet by %s", snippet.OwnerName),
		[]templates.Breadcrumb{{Name: snippet.OwnerName, Url: snippet.OwnerUrl}},
	)
	baseData.OpenGraphItems = opengraph // NOTE(asaf): We're overriding the defaults on purpose.
	var res ResponseData
	err = res.WriteTemplate("snippet.html", SnippetData{
		BaseData: baseData,
		Snippet:  snippet,
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render snippet template"))
	}
	return res
}

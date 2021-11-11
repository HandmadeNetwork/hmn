package website

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"git.handmade.network/hmn/hmn/src/db"
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

	s, err := FetchSnippet(c.Context(), c.Conn, c.CurrentUser, snippetId, SnippetQuery{})
	if err != nil {
		if errors.Is(err, db.NotFound) {
			return FourOhFour(c)
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippet"))
		}
	}
	c.Perf.EndBlock()

	snippet := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Tags, s.Owner, c.Theme)

	opengraph := []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade.Network"},
		{Property: "og:type", Value: "article"},
		{Property: "og:url", Value: snippet.Url},
		{Property: "og:title", Value: fmt.Sprintf("Snippet by %s", snippet.OwnerName)},
		{Property: "og:description", Value: string(snippet.Description)},
	}

	if len(snippet.EmbedMedia) > 0 {
		media := snippet.EmbedMedia[0]

		switch media.Type {
		case templates.TimelineItemMediaTypeImage:
			opengraph = append(opengraph,
				templates.OpenGraphItem{Property: "og:image", Value: media.AssetUrl},
				templates.OpenGraphItem{Property: "og:image:width", Value: strconv.Itoa(media.Width)},
				templates.OpenGraphItem{Property: "og:image:height", Value: strconv.Itoa(media.Height)},
				templates.OpenGraphItem{Property: "og:image:type", Value: media.MimeType},
				templates.OpenGraphItem{Name: "twitter:card", Value: "summary_large_image"},
			)
		case templates.TimelineItemMediaTypeVideo:
			opengraph = append(opengraph,
				templates.OpenGraphItem{Property: "og:video", Value: media.AssetUrl},
				templates.OpenGraphItem{Property: "og:video:width", Value: strconv.Itoa(media.Width)},
				templates.OpenGraphItem{Property: "og:video:height", Value: strconv.Itoa(media.Height)},
				templates.OpenGraphItem{Property: "og:video:type", Value: media.MimeType},
				templates.OpenGraphItem{Name: "twitter:card", Value: "player"},
			)
		case templates.TimelineItemMediaTypeAudio:
			opengraph = append(opengraph,
				templates.OpenGraphItem{Property: "og:audio", Value: media.AssetUrl},
				templates.OpenGraphItem{Property: "og:audio:type", Value: media.MimeType},
				templates.OpenGraphItem{Name: "twitter:card", Value: "player"},
			)
		}
		opengraph = append(opengraph, media.ExtraOpenGraphItems...)
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

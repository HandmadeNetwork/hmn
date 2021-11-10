package website

import (
	"html/template"
	"math"
	"net/http"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

type LandingTemplateData struct {
	templates.BaseData

	NewsPost             *templates.TimelineItem
	TimelineItems        []templates.TimelineItem
	Pagination           templates.Pagination
	ShowcaseTimelineJson string

	ManifestoUrl   string
	FeedUrl        string
	PodcastUrl     string
	StreamsUrl     string
	ShowcaseUrl    string
	AtomFeedUrl    string
	MarkAllReadUrl string

	WheelJamUrl string
}

func Index(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	var timelineItems []templates.TimelineItem

	numPosts, err := CountPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
		ThreadTypes: feedThreadTypes,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	numPages := int(math.Ceil(float64(numPosts) / feedPostsPerPage))

	page, numPages, ok := getPageInfo("1", numPosts, feedPostsPerPage)
	if !ok {
		return c.Redirect(hmnurl.BuildFeed(), http.StatusSeeOther)
	}

	pagination := templates.Pagination{
		Current: page,
		Total:   numPages,

		FirstUrl:    hmnurl.BuildFeed(),
		LastUrl:     hmnurl.BuildFeedWithPage(numPages),
		NextUrl:     hmnurl.BuildFeedWithPage(utils.IntClamp(1, page+1, numPages)),
		PreviousUrl: hmnurl.BuildFeedWithPage(utils.IntClamp(1, page-1, numPages)),
	}

	// This is essentially an alternate for feed page 1.
	posts, err := FetchPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
		ThreadTypes:    feedThreadTypes,
		Limit:          feedPostsPerPage,
		SortDescending: true,
	})
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch latest posts")
	}
	for _, p := range posts {
		item := PostToTimelineItem(UrlContextForProject(&p.Project), lineageBuilder, &p.Post, &p.Thread, p.Author, c.Theme)
		if p.Thread.Type == models.ThreadTypeProjectBlogPost && p.Post.ID == p.Thread.FirstID {
			// blog post
			item.Description = template.HTML(p.CurrentVersion.TextParsed)
			item.TruncateDescription = true
		}
		timelineItems = append(timelineItems, item)
	}

	c.Perf.StartBlock("SQL", "Get news")
	newsThreads, err := FetchThreads(c.Context(), c.Conn, c.CurrentUser, ThreadsQuery{
		ProjectIDs:  []int{models.HMNProjectID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
		Limit:       1,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch news post"))
	}
	var newsPostItem *templates.TimelineItem
	if len(newsThreads) > 0 {
		t := newsThreads[0]
		item := PostToTimelineItem(UrlContextForProject(&t.Project), lineageBuilder, &t.FirstPost, &t.Thread, t.FirstPostAuthor, c.Theme)
		item.OwnerAvatarUrl = ""
		item.Breadcrumbs = nil
		item.TypeTitle = ""
		item.AllowTitleWrap = true
		item.Description = template.HTML(t.FirstPostCurrentVersion.TextParsed)
		item.TruncateDescription = true
		newsPostItem = &item
	}
	c.Perf.EndBlock()

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
			NOT snippet.is_jam
		ORDER BY snippet.when DESC
		LIMIT 40
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
		if timelineItem.CanShowcase {
			showcaseItems = append(showcaseItems, timelineItem)
		}
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SHOWCASE", "Convert to json")
	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)
	c.Perf.EndBlock()

	baseData := getBaseData(c, "", nil)
	baseData.BodyClasses = append(baseData.BodyClasses, "hmdev", "landing") // TODO: Is "hmdev" necessary any more?
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    "A community of programmers committed to producing quality software through deeper understanding.",
	})

	var res ResponseData
	err = res.WriteTemplate("landing.html", LandingTemplateData{
		BaseData: baseData,

		NewsPost:             newsPostItem,
		TimelineItems:        timelineItems,
		Pagination:           pagination,
		ShowcaseTimelineJson: showcaseJson,

		ManifestoUrl:   hmnurl.BuildManifesto(),
		FeedUrl:        hmnurl.BuildFeed(),
		PodcastUrl:     hmnurl.BuildPodcast(),
		StreamsUrl:     hmnurl.BuildStreams(),
		ShowcaseUrl:    hmnurl.BuildShowcase(),
		AtomFeedUrl:    hmnurl.BuildAtomFeed(),
		MarkAllReadUrl: hmnurl.HMNProjectContext.BuildForumMarkRead(0),

		WheelJamUrl: hmnurl.BuildJamIndex(),
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render landing page template"))
	}

	return res
}

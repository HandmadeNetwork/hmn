package website

import (
	"html/template"
	"math"
	"net/http"

	"git.handmade.network/hmn/hmn/src/hmndata"
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

	JamUrl string
}

func Index(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	var timelineItems []templates.TimelineItem

	numPosts, err := hmndata.CountPosts(c.Context(), c.Conn, c.CurrentUser, hmndata.PostsQuery{
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
	posts, err := hmndata.FetchPosts(c.Context(), c.Conn, c.CurrentUser, hmndata.PostsQuery{
		ThreadTypes:    feedThreadTypes,
		Limit:          feedPostsPerPage,
		SortDescending: true,
	})
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch latest posts")
	}
	for _, p := range posts {
		item := PostToTimelineItem(hmndata.UrlContextForProject(&p.Project), lineageBuilder, &p.Post, &p.Thread, p.Author, c.Theme)
		if p.Thread.Type == models.ThreadTypeProjectBlogPost && p.Post.ID == p.Thread.FirstID {
			// blog post
			item.Description = template.HTML(p.CurrentVersion.TextParsed)
			item.TruncateDescription = true
		}
		timelineItems = append(timelineItems, item)
	}

	c.Perf.StartBlock("SQL", "Get news")
	newsThreads, err := hmndata.FetchThreads(c.Context(), c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
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
		item := PostToTimelineItem(hmndata.UrlContextForProject(&t.Project), lineageBuilder, &t.FirstPost, &t.Thread, t.FirstPostAuthor, c.Theme)
		item.OwnerAvatarUrl = ""
		item.Breadcrumbs = nil
		item.TypeTitle = ""
		item.AllowTitleWrap = true
		item.Description = template.HTML(t.FirstPostCurrentVersion.TextParsed)
		item.TruncateDescription = true
		newsPostItem = &item
	}
	c.Perf.EndBlock()

	snippets, err := hmndata.FetchSnippets(c.Context(), c.Conn, c.CurrentUser, hmndata.SnippetQuery{
		Limit: 40,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets"))
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

		JamUrl: hmnurl.BuildJamIndex(),
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render landing page template"))
	}

	return res
}

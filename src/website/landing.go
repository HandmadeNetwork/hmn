package website

import (
	"html/template"
	"net/http"

	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

type LandingTemplateData struct {
	templates.BaseData

	NewsPost       *templates.TimelineItem
	FollowingItems []templates.TimelineItem
	FeaturedItems  []templates.TimelineItem
	RecentItems    []templates.TimelineItem
	NewsItems      []templates.TimelineItem

	ManifestoUrl   string
	PodcastUrl     string
	AtomFeedUrl    string
	MarkAllReadUrl string

	JamUrl                             string
	JamDaysUntilStart, JamDaysUntilEnd int

	HMSDaysUntilStart, HMSDaysUntilEnd           int
	HMBostonDaysUntilStart, HMBostonDaysUntilEnd int
}

func Index(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c, c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	var timelineItems []templates.TimelineItem

	// This is essentially an alternate for feed page 1.
	posts, err := hmndata.FetchPosts(c, c.Conn, c.CurrentUser, hmndata.PostsQuery{
		ThreadTypes:    feedThreadTypes,
		Limit:          feedPostsPerPage,
		SortDescending: true,
	})
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch latest posts")
	}
	for _, p := range posts {
		if p.Project.IsHMN() {
			continue // ignore news posts et. al.
		}

		item := PostToTimelineItem(hmndata.UrlContextForProject(&p.Project), lineageBuilder, &p.Post, &p.Thread, p.Author, c.Theme)
		if p.Thread.Type == models.ThreadTypeProjectBlogPost && p.Post.ID == p.Thread.FirstID {
			// blog post
			item.Description = template.HTML(p.CurrentVersion.TextParsed)
			item.TruncateDescription = true
		}
		timelineItems = append(timelineItems, item)
	}

	c.Perf.StartBlock("SQL", "Get news")
	newsThreads, err := hmndata.FetchThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
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
		item.Breadcrumbs = nil
		item.TypeTitle = ""
		item.AllowTitleWrap = true
		item.Description = template.HTML(t.FirstPostCurrentVersion.TextParsed)
		item.TruncateDescription = true
		newsPostItem = &item
	}
	c.Perf.EndBlock()

	baseData := getBaseData(c, "", nil)
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    "A community of low-level programmers with high-level goals, working to correct the course of the software industry.",
	})

	var res ResponseData
	err = res.WriteTemplate("landing.html", LandingTemplateData{
		BaseData: baseData,

		NewsPost:       newsPostItem,
		FollowingItems: timelineItems,

		ManifestoUrl:   hmnurl.BuildManifesto(),
		PodcastUrl:     hmnurl.BuildPodcast(),
		AtomFeedUrl:    hmnurl.BuildAtomFeed(),
		MarkAllReadUrl: hmnurl.HMNProjectContext.BuildForumMarkRead(0),

		JamUrl:            hmnurl.BuildJamIndex2024_Learning(),
		JamDaysUntilStart: daysUntil(hmndata.LJ2024.StartTime),
		JamDaysUntilEnd:   daysUntil(hmndata.LJ2024.EndTime),

		HMSDaysUntilStart: daysUntil(hmndata.HMS2024.StartTime),
		HMSDaysUntilEnd:   daysUntil(hmndata.HMS2024.EndTime),

		HMBostonDaysUntilStart: daysUntil(hmndata.HMBoston2024.StartTime),
		HMBostonDaysUntilEnd:   daysUntil(hmndata.HMBoston2024.EndTime),
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render landing page template"))
	}

	return res
}

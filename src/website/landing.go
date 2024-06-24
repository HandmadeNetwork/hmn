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

func Index(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c, c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	type LandingTemplateData struct {
		templates.BaseData

		FollowingItems []templates.TimelineItem
		FeaturedItems  []templates.TimelineItem
		RecentItems    []templates.TimelineItem
		NewsItems      []templates.TimelineItem

		UserProjects []templates.Project
		Following    []templates.Follow

		ManifestoUrl   string
		AboutUrl       string
		PodcastUrl     string
		AtomFeedUrl    string
		MarkAllReadUrl string
		NewProjectUrl  string

		JamUrl                             string
		JamDaysUntilStart, JamDaysUntilEnd int

		HMSDaysUntilStart, HMSDaysUntilEnd           int
		HMBostonDaysUntilStart, HMBostonDaysUntilEnd int
	}

	var err error
	var followingItems []templates.TimelineItem
	var featuredItems []templates.TimelineItem
	var recentItems []templates.TimelineItem
	var newsItems []templates.TimelineItem

	if c.CurrentUser != nil {
		followingItems, err = FetchFollowTimelineForUser(c, c.Conn, c.CurrentUser)
		if err != nil {
			c.Logger.Warn().Err(err).Msg("failed to fetch following feed")
		}
	}

	featuredProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		FeaturedOnly: true,
	})
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch featured projects")
	}
	var featuredProjectIDs []int
	for _, p := range featuredProjects {
		featuredProjectIDs = append(featuredProjectIDs, p.Project.ID)
	}
	featuredItems, err = FetchTimeline(c, c.Conn, c.CurrentUser, TimelineQuery{
		ProjectIDs: featuredProjectIDs,
		Limit:      100,
	})
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch featured feed")
	}

	recentItems, err = FetchTimeline(c, c.Conn, c.CurrentUser, TimelineQuery{
		Limit: 100,
	})
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch recent feed")
	}

	newsThreads, err := hmndata.FetchThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
		ProjectIDs:     []int{models.HMNProjectID},
		ThreadTypes:    []models.ThreadType{models.ThreadTypeProjectBlogPost},
		Limit:          100,
		OrderByCreated: true,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch news threads"))
	}
	for _, t := range newsThreads {
		item := PostToTimelineItem(c.UrlContext, lineageBuilder, &t.FirstPost, &t.Thread, t.FirstPostAuthor)
		item.Breadcrumbs = nil
		item.TypeTitle = ""
		item.Description = template.HTML(t.FirstPostCurrentVersion.TextParsed)
		item.AllowTitleWrap = true
		item.TruncateDescription = true
		newsItems = append(newsItems, item)
	}

	var projects []templates.Project
	if c.CurrentUser != nil {
		projectsDb, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
		})
		if err != nil {
			c.Logger.Warn().Err(err).Msg("failed to fetch user projects")
		}

		for _, p := range projectsDb {
			projects = append(projects, templates.ProjectAndStuffToTemplate(&p))
		}
	}

	var follows []templates.Follow
	if c.CurrentUser != nil {
		follows, err = FetchFollows(c, c.Conn, c.CurrentUser, c.CurrentUser.ID)
		if err != nil {
			c.Logger.Warn().Err(err).Msg("failed to fetch user follows")
		}
	}

	baseData := getBaseData(c, "", nil)
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    "A community of low-level programmers with high-level goals, working to correct the course of the software industry.",
	})

	var res ResponseData
	err = res.WriteTemplate("landing.html", LandingTemplateData{
		BaseData: baseData,

		FollowingItems: followingItems,
		FeaturedItems:  featuredItems,
		RecentItems:    recentItems,
		NewsItems:      newsItems,

		UserProjects: projects,
		Following:    follows,

		ManifestoUrl:   hmnurl.BuildManifesto(),
		AboutUrl:       hmnurl.BuildAbout(),
		PodcastUrl:     hmnurl.BuildPodcast(),
		AtomFeedUrl:    hmnurl.BuildAtomFeed(),
		MarkAllReadUrl: hmnurl.HMNProjectContext.BuildForumMarkRead(0),
		NewProjectUrl:  hmnurl.BuildProjectNew(),

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

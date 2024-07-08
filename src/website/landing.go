package website

import (
	"html/template"
	"net/http"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

func Index(c *RequestContext) ResponseData {
	const maxPostsPerTab = 50
	const maxNewsPosts = 10

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
		GuidelinesUrl  string
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
		followingItems, err = FetchFollowTimelineForUser(
			c, c.Conn,
			c.CurrentUser,
			lineageBuilder,
			FollowTimelineQuery{
				Limit: maxPostsPerTab,
			},
		)
		if err != nil {
			c.Logger.Warn().Err(err).Msg("failed to fetch following feed")
		}
	}

	featuredProjectIDs, err := db.QueryScalar[int](c, c.Conn,
		`
		SELECT id
		FROM project
		WHERE featured = true
		`,
	)
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch featured projects")
	}
	featuredUserIDs, err := db.QueryScalar[int](c, c.Conn,
		`
		SELECT id
		FROM hmn_user
		WHERE featured = true
		`,
	)
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch featured users")
	}
	featuredItems, err = FetchTimeline(c, c.Conn, c.CurrentUser, lineageBuilder, hmndata.TimelineQuery{
		ProjectIDs: featuredProjectIDs,
		OwnerIDs:   featuredUserIDs,
		Limit:      maxPostsPerTab,

		SkipPosts: true,
	})
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch featured feed")
	}

	recentItems, err = FetchTimeline(c, c.Conn, c.CurrentUser, lineageBuilder, hmndata.TimelineQuery{
		Limit: maxPostsPerTab,
	})
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to fetch recent feed")
	}

	newsThreads, err := hmndata.FetchThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
		ProjectIDs:     []int{models.HMNProjectID},
		ThreadTypes:    []models.ThreadType{models.ThreadTypeProjectBlogPost},
		Limit:          maxNewsPosts,
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
		item.Unread = t.Unread
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
		GuidelinesUrl:  hmnurl.BuildCommunicationGuidelines(),
		AtomFeedUrl:    hmnurl.BuildAtomFeed(),
		MarkAllReadUrl: hmnurl.HMNProjectContext.BuildForumMarkRead(0),
		NewProjectUrl:  hmnurl.BuildProjectNew(),

		JamUrl:            hmnurl.BuildJamIndex2024_Learning(),
		JamDaysUntilStart: utils.DaysUntil(hmndata.LJ2024.StartTime),
		JamDaysUntilEnd:   utils.DaysUntil(hmndata.LJ2024.EndTime),

		HMSDaysUntilStart: utils.DaysUntil(hmndata.HMS2024.StartTime),
		HMSDaysUntilEnd:   utils.DaysUntil(hmndata.HMS2024.EndTime),

		HMBostonDaysUntilStart: utils.DaysUntil(hmndata.HMBoston2024.StartTime),
		HMBostonDaysUntilEnd:   utils.DaysUntil(hmndata.HMBoston2024.EndTime),
	}, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render landing page template"))
	}

	return res
}

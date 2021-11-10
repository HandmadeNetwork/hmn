package website

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

type FeedData struct {
	templates.BaseData

	Posts          []templates.PostListItem
	Pagination     templates.Pagination
	AtomFeedUrl    string
	MarkAllReadUrl string
}

const feedPostsPerPage = 30

var feedThreadTypes = []models.ThreadType{
	models.ThreadTypeForumPost,
	models.ThreadTypeProjectBlogPost,
}

func Feed(c *RequestContext) ResponseData {
	numPosts, err := CountPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
		ThreadTypes: feedThreadTypes,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	numPages := int(math.Ceil(float64(numPosts) / feedPostsPerPage))

	page, numPages, ok := getPageInfo(c.PathParams["page"], numPosts, feedPostsPerPage)
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

	posts, err := fetchAllPosts(c, (page-1)*feedPostsPerPage, feedPostsPerPage)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed posts"))
	}

	baseData := getBaseDataAutocrumb(c, "Feed")
	baseData.BodyClasses = append(baseData.BodyClasses, "feed")

	var res ResponseData
	res.MustWriteTemplate("feed.html", FeedData{
		BaseData: baseData,

		AtomFeedUrl:    hmnurl.BuildAtomFeed(),
		MarkAllReadUrl: c.UrlContext.BuildForumMarkRead(0),
		Posts:          posts,
		Pagination:     pagination,
	}, c.Perf)
	return res
}

type FeedType int

const (
	FeedTypeAll = iota
	FeedTypeProjects
	FeedTypeShowcase
)

// NOTE(asaf): UUID values copied from old website
var (
	FeedIDAll      = "urn:uuid:1084fd28-993a-4961-9011-39ddeaeb3711"
	FeedIDProjects = "urn:uuid:cfad0d50-cbcf-11e7-82d7-db1d52543cc7"
	FeedIDShowcase = "urn:uuid:37d29027-2892-5a21-b521-951246c7aa46"
)

type AtomFeedData struct {
	Title    string
	Subtitle string

	HomepageUrl string
	AtomFeedUrl string
	FeedUrl     string

	CopyrightStatement string
	SiteVersion        string
	Updated            time.Time
	FeedID             string

	FeedType FeedType
	Posts    []templates.PostListItem
	Projects []templates.Project
	Snippets []templates.TimelineItem
}

func AtomFeed(c *RequestContext) ResponseData {
	itemsPerFeed := 25 // NOTE(asaf): Copied from old website

	feedData := AtomFeedData{
		HomepageUrl: hmnurl.BuildHomepage(),

		CopyrightStatement: fmt.Sprintf("Copyright (C) 2014-%d Handmade.Network and its contributors", time.Now().Year()),
		SiteVersion:        "2.0",
	}

	feedType, hasType := c.PathParams["feedtype"]
	if !hasType || len(feedType) == 0 {
		feedData.Title = "New Threads, Blog Posts, Replies and Comments | Site-wide | Handmade.Network"
		feedData.Subtitle = feedData.Title
		feedData.FeedType = FeedTypeAll
		feedData.FeedID = FeedIDAll
		feedData.AtomFeedUrl = hmnurl.BuildAtomFeed()
		feedData.FeedUrl = hmnurl.BuildFeed()

		posts, err := fetchAllPosts(c, 0, itemsPerFeed)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed posts"))
		}
		feedData.Posts = posts

		updated := time.Now()
		if len(posts) > 0 {
			updated = posts[0].Date
		}
		feedData.Updated = updated
	} else {
		switch strings.ToLower(feedType) {
		case "projects":
			feedData.Title = "New Projects | Site-wide | Handmade.Network"
			feedData.Subtitle = feedData.Title
			feedData.FeedType = FeedTypeProjects
			feedData.FeedID = FeedIDProjects
			feedData.AtomFeedUrl = hmnurl.BuildAtomFeedForProjects()
			feedData.FeedUrl = hmnurl.BuildProjectIndex(1)

			c.Perf.StartBlock("SQL", "Fetching projects")
			type projectResult struct {
				Project models.Project `db:"project"`
			}
			_, hasAll := c.Req.URL.Query()["all"]
			if hasAll {
				itemsPerFeed = 100000
			}
			projects, err := db.Query(c.Context(), c.Conn, projectResult{},
				`
				SELECT $columns
				FROM
					handmade_project AS project
				WHERE
					project.lifecycle = ANY($1)
					AND NOT project.hidden
				ORDER BY date_approved DESC
				LIMIT $2
				`,
				models.VisibleProjectLifecycles,
				itemsPerFeed,
			)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed projects"))
			}
			var projectIds []int
			projectMap := make(map[int]int) // map[project id]index in slice
			for _, p := range projects.ToSlice() {
				project := p.(*projectResult).Project
				templateProject := templates.ProjectToTemplate(&project, UrlContextForProject(&project).BuildHomepage(), c.Theme)
				templateProject.UUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(templateProject.Url)).URN()

				projectIds = append(projectIds, project.ID)
				feedData.Projects = append(feedData.Projects, templateProject)
				projectMap[project.ID] = len(feedData.Projects) - 1
			}
			c.Perf.EndBlock()

			c.Perf.StartBlock("SQL", "Fetching project owners")
			type ownerResult struct {
				User      models.User `db:"auth_user"`
				ProjectID int         `db:"uproj.project_id"`
			}
			owners, err := db.Query(c.Context(), c.Conn, ownerResult{},
				`
				SELECT $columns
				FROM
					handmade_user_projects AS uproj
					JOIN auth_user ON uproj.user_id = auth_user.id
				WHERE
					uproj.project_id = ANY($1)
				`,
				projectIds,
			)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed projects owners"))
			}
			for _, res := range owners.ToSlice() {
				owner := res.(*ownerResult)
				templateProject := &feedData.Projects[projectMap[owner.ProjectID]]
				templateProject.Owners = append(templateProject.Owners, templates.UserToTemplate(&owner.User, ""))
			}
			c.Perf.EndBlock()
			updated := time.Now()
			if len(feedData.Projects) > 0 {
				updated = feedData.Projects[0].DateApproved
			}
			feedData.Updated = updated
		case "showcase":
			feedData.Title = "Showcase | Site-wide | Handmade.Network"
			feedData.Subtitle = feedData.Title
			feedData.FeedType = FeedTypeShowcase
			feedData.FeedID = FeedIDShowcase
			feedData.AtomFeedUrl = hmnurl.BuildAtomFeedForShowcase()
			feedData.FeedUrl = hmnurl.BuildShowcase()

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
				LIMIT $1
				`,
				itemsPerFeed,
			)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets"))
			}
			snippetQuerySlice := snippetQueryResult.ToSlice()
			for _, s := range snippetQuerySlice {
				row := s.(*snippetQuery)
				timelineItem := SnippetToTimelineItem(&row.Snippet, row.Asset, row.DiscordMessage, &row.Owner, c.Theme)
				feedData.Snippets = append(feedData.Snippets, timelineItem)
			}
			c.Perf.EndBlock()
			updated := time.Now()
			if len(feedData.Snippets) > 0 {
				updated = feedData.Snippets[0].Date
			}
			feedData.Updated = updated
		default:
			return FourOhFour(c)
		}
	}

	var res ResponseData
	res.MustWriteTemplate("atom.xml", feedData, c.Perf)
	return res
}

func fetchAllPosts(c *RequestContext, offset int, limit int) ([]templates.PostListItem, error) {
	postsAndStuff, err := FetchPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
		ThreadTypes:    feedThreadTypes,
		Limit:          limit,
		Offset:         offset,
		SortDescending: true,
	})
	if err != nil {
		return nil, err
	}
	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	c.Perf.StartBlock("FEED", "Build post items")
	var postItems []templates.PostListItem
	for _, postAndStuff := range postsAndStuff {
		postItem := MakePostListItem(
			lineageBuilder,
			&postAndStuff.Project,
			&postAndStuff.Thread,
			&postAndStuff.Post,
			postAndStuff.Author,
			postAndStuff.Unread,
			true,
			c.Theme,
		)

		postItem.UUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(postItem.Url)).URN()
		postItem.LastEditDate = postAndStuff.CurrentVersion.Date

		postItems = append(postItems, postItem)
	}
	c.Perf.EndBlock()

	return postItems, nil
}

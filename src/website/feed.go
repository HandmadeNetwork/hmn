package website

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/google/uuid"
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
	numPosts, err := hmndata.CountPosts(c.Context(), c.Conn, c.CurrentUser, hmndata.PostsQuery{
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
			_, hasAll := c.Req.URL.Query()["all"]
			if hasAll {
				itemsPerFeed = 100000
			}
			projectsAndStuff, err := hmndata.FetchProjects(c.Context(), c.Conn, nil, hmndata.ProjectsQuery{
				Lifecycles: models.VisibleProjectLifecycles,
				Limit:      itemsPerFeed,
				Types:      hmndata.OfficialProjects,
				OrderBy:    "date_approved DESC",
			})
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed projects"))
			}
			for _, p := range projectsAndStuff {
				templateProject := templates.ProjectToTemplate(&p.Project, hmndata.UrlContextForProject(&p.Project).BuildHomepage())
				templateProject.UUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(templateProject.Url)).URN()
				for _, owner := range p.Owners {
					templateProject.Owners = append(templateProject.Owners, templates.UserToTemplate(owner, ""))
				}

				feedData.Projects = append(feedData.Projects, templateProject)
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

			snippets, err := hmndata.FetchSnippets(c.Context(), c.Conn, c.CurrentUser, hmndata.SnippetQuery{
				Limit: itemsPerFeed,
			})
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets"))
			}
			for _, s := range snippets {
				timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Tags, s.Owner, c.Theme)
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
	postsAndStuff, err := hmndata.FetchPosts(c.Context(), c.Conn, c.CurrentUser, hmndata.PostsQuery{
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

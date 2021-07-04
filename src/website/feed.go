package website

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
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

func Feed(c *RequestContext) ResponseData {
	const postsPerPage = 30

	c.Perf.StartBlock("SQL", "Count posts")
	numPosts, err := db.QueryInt(c.Context(), c.Conn,
		`
		SELECT COUNT(*)
		FROM
			handmade_post AS post
		WHERE
			post.category_kind = ANY ($1)
			AND deleted = FALSE
			AND post.thread_id IS NOT NULL
		`,
		[]models.CategoryKind{models.CatKindForum, models.CatKindBlog, models.CatKindWiki, models.CatKindLibraryResource},
	)
	c.Perf.EndBlock()
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get count of feed posts"))
	}

	numPages := int(math.Ceil(float64(numPosts) / postsPerPage))

	page := 1
	pageString, hasPage := c.PathParams["page"]
	if hasPage && pageString != "" {
		if pageParsed, err := strconv.Atoi(pageString); err == nil {
			page = pageParsed
		} else {
			return c.Redirect(hmnurl.BuildFeed(), http.StatusSeeOther)
		}
	}
	if page < 1 || numPages < page {
		return c.Redirect(hmnurl.BuildFeedWithPage(utils.IntClamp(1, page, numPages)), http.StatusSeeOther)
	}

	howManyPostsToSkip := (page - 1) * postsPerPage

	pagination := templates.Pagination{
		Current: page,
		Total:   numPages,

		FirstUrl:    hmnurl.BuildFeed(),
		LastUrl:     hmnurl.BuildFeedWithPage(numPages),
		NextUrl:     hmnurl.BuildFeedWithPage(utils.IntClamp(1, page+1, numPages)),
		PreviousUrl: hmnurl.BuildFeedWithPage(utils.IntClamp(1, page-1, numPages)),
	}

	var currentUserId *int
	if c.CurrentUser != nil {
		currentUserId = &c.CurrentUser.ID
	}

	c.Perf.StartBlock("SQL", "Fetch category tree")
	categoryTree := models.GetFullCategoryTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeCategoryLineageBuilder(categoryTree)
	c.Perf.EndBlock()

	posts, err := fetchAllPosts(c, lineageBuilder, currentUserId, howManyPostsToSkip, postsPerPage)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed posts"))
	}

	baseData := getBaseData(c)
	baseData.BodyClasses = append(baseData.BodyClasses, "feed")

	var res ResponseData
	err = res.WriteTemplate("feed.html", FeedData{
		BaseData: baseData,

		AtomFeedUrl:    hmnurl.BuildAtomFeed(),
		MarkAllReadUrl: hmnurl.BuildMarkRead(0),
		Posts:          posts,
		Pagination:     pagination,
	}, c.Perf)
	if err != nil {
		panic(err)
	}

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

		c.Perf.StartBlock("SQL", "Fetch category tree")
		categoryTree := models.GetFullCategoryTree(c.Context(), c.Conn)
		lineageBuilder := models.MakeCategoryLineageBuilder(categoryTree)
		c.Perf.EndBlock()

		posts, err := fetchAllPosts(c, lineageBuilder, nil, 0, itemsPerFeed)
		if err != nil {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed posts"))
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
			projects, err := db.Query(c.Context(), c.Conn, projectResult{},
				`
				SELECT $columns
				FROM
					handmade_project AS project
				WHERE
					project.lifecycle = ANY($1)
					AND project.flags = 0
				ORDER BY date_approved DESC
				LIMIT $2
				`,
				models.VisibleProjectLifecycles,
				itemsPerFeed,
			)
			if err != nil {
				return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed projects"))
			}
			var projectIds []int
			projectMap := make(map[int]*templates.Project)
			for _, p := range projects.ToSlice() {
				project := p.(*projectResult).Project
				templateProject := templates.ProjectToTemplate(&project, c.Theme)
				templateProject.UUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(templateProject.Url)).URN()

				projectIds = append(projectIds, project.ID)
				projectMap[project.ID] = &templateProject
				feedData.Projects = append(feedData.Projects, templateProject)
			}
			c.Perf.EndBlock()

			c.Perf.StartBlock("SQL", "Fetching project owners")
			type ownerResult struct {
				User      models.User `db:"auth_user"`
				ProjectID int         `db:"project_groups.project_id"`
			}
			owners, err := db.Query(c.Context(), c.Conn, ownerResult{},
				`
				SELECT $columns
				FROM
					auth_user 
					INNER JOIN auth_user_groups AS user_groups ON auth_user.id = user_groups.user_id
					INNER JOIN handmade_project_groups AS project_groups ON user_groups.group_id = project_groups.group_id
				WHERE
					project_groups.project_id = ANY($1)
				`,
				projectIds,
			)
			if err != nil {
				return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed projects owners"))
			}
			for _, res := range owners.ToSlice() {
				owner := res.(*ownerResult)
				templateProject := projectMap[owner.ProjectID]
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
				ORDER BY snippet.when DESC
				LIMIT $1
				`,
				itemsPerFeed,
			)
			if err != nil {
				return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets"))
			}
			snippetQuerySlice := snippetQueryResult.ToSlice()
			for _, s := range snippetQuerySlice {
				row := s.(*snippetQuery)
				timelineItem := SnippetToTimelineItem(&row.Snippet, row.Asset, row.DiscordMessage, &row.Owner, c.Theme)
				timelineItem.UUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(timelineItem.Url)).URN()
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
	err := res.WriteTemplate("atom.xml", feedData, c.Perf)
	if err != nil {
		panic(err)
	}

	return res
}

func fetchAllPosts(c *RequestContext, lineageBuilder *models.CategoryLineageBuilder, currentUserID *int, offset int, limit int) ([]templates.PostListItem, error) {
	c.Perf.StartBlock("SQL", "Fetch posts")
	type feedPostQuery struct {
		Post               models.Post             `db:"post"`
		PostVersion        models.PostVersion      `db:"version"`
		Thread             models.Thread           `db:"thread"`
		Cat                models.Category         `db:"cat"`
		Proj               models.Project          `db:"proj"`
		LibraryResource    *models.LibraryResource `db:"lib_resource"`
		User               models.User             `db:"auth_user"`
		ThreadLastReadTime *time.Time              `db:"tlri.lastread"`
		CatLastReadTime    *time.Time              `db:"clri.lastread"`
	}
	posts, err := db.Query(c.Context(), c.Conn, feedPostQuery{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			JOIN handmade_postversion AS version ON version.id = post.current_id
			JOIN handmade_thread AS thread ON thread.id = post.thread_id
			JOIN handmade_category AS cat ON cat.id = post.category_id
			JOIN handmade_project AS proj ON proj.id = post.project_id
			LEFT JOIN handmade_threadlastreadinfo AS tlri ON (
				tlri.thread_id = post.thread_id
				AND tlri.user_id = $1
			)
			LEFT JOIN handmade_categorylastreadinfo AS clri ON (
				clri.category_id = post.category_id
				AND clri.user_id = $1
			)
			LEFT JOIN auth_user ON post.author_id = auth_user.id
			LEFT JOIN handmade_libraryresource as lib_resource ON lib_resource.category_id = post.category_id
		WHERE
			post.category_kind = ANY ($2)
			AND post.deleted = FALSE
			AND post.thread_id IS NOT NULL
		ORDER BY postdate DESC
		LIMIT $3 OFFSET $4
		`,
		currentUserID,
		[]models.CategoryKind{models.CatKindForum, models.CatKindBlog, models.CatKindWiki, models.CatKindLibraryResource},
		limit,
		offset,
	)
	c.Perf.EndBlock()
	if err != nil {
		return nil, err
	}

	c.Perf.StartBlock("FEED", "Build post items")
	var postItems []templates.PostListItem
	for _, iPostResult := range posts.ToSlice() {
		postResult := iPostResult.(*feedPostQuery)

		hasRead := false
		if postResult.ThreadLastReadTime != nil && postResult.ThreadLastReadTime.After(postResult.Post.PostDate) {
			hasRead = true
		} else if postResult.CatLastReadTime != nil && postResult.CatLastReadTime.After(postResult.Post.PostDate) {
			hasRead = true
		}

		postItem := MakePostListItem(
			lineageBuilder,
			&postResult.Proj,
			&postResult.Thread,
			&postResult.Post,
			&postResult.User,
			postResult.LibraryResource,
			!hasRead,
			true,
			c.Theme,
		)

		postItem.UUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(postItem.Url)).URN()
		postItem.LastEditDate = postResult.PostVersion.Date

		postItems = append(postItems, postItem)
	}
	c.Perf.EndBlock()

	return postItems, nil
}

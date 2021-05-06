package website

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

type FeedData struct {
	templates.BaseData

	Posts      []templates.PostListItem
	Pagination templates.Pagination
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

	c.Perf.StartBlock("SQL", "Fetch posts")
	type feedPostQuery struct {
		Post               models.Post     `db:"post"`
		Thread             models.Thread   `db:"thread"`
		Cat                models.Category `db:"cat"`
		Proj               models.Project  `db:"proj"`
		User               models.User     `db:"auth_user"`
		ThreadLastReadTime *time.Time      `db:"tlri.lastread"`
		CatLastReadTime    *time.Time      `db:"clri.lastread"`
	}
	posts, err := db.Query(c.Context(), c.Conn, feedPostQuery{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			JOIN handmade_thread AS thread ON thread.id = post.thread_id
			JOIN handmade_category AS cat ON cat.id = post.category_id
			JOIN handmade_project AS proj ON proj.id = post.project_id
			LEFT OUTER JOIN handmade_threadlastreadinfo AS tlri ON (
				tlri.thread_id = post.thread_id
				AND tlri.user_id = $1
			)
			LEFT OUTER JOIN handmade_categorylastreadinfo AS clri ON (
				clri.category_id = post.category_id
				AND clri.user_id = $1
			)
			LEFT OUTER JOIN auth_user ON post.author_id = auth_user.id
		WHERE
			post.category_kind = ANY ($2)
			AND post.deleted = FALSE
			AND post.thread_id IS NOT NULL
		ORDER BY postdate DESC
		LIMIT $3 OFFSET $4
		`,
		currentUserId,
		[]models.CategoryKind{models.CatKindForum, models.CatKindBlog, models.CatKindWiki, models.CatKindLibraryResource},
		postsPerPage,
		howManyPostsToSkip,
	)
	c.Perf.EndBlock()
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed posts"))
	}

	c.Perf.StartBlock("SQL", "Fetch category tree")
	categoryTree := models.GetFullCategoryTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeCategoryLineageBuilder(categoryTree)
	c.Perf.EndBlock()

	categoryUrlCache := make(map[int]string)
	getCategoryUrl := func(projectSlug string, cat *models.Category) string {
		_, ok := categoryUrlCache[cat.ID]
		if !ok {
			lineageNames := lineageBuilder.GetLineageSlugs(cat.ID)
			switch cat.Kind {
			case models.CatKindForum:
				categoryUrlCache[cat.ID] = hmnurl.BuildForumCategory(projectSlug, lineageNames[1:], 1)
				// TODO(asaf): Add more kinds!!!
			default:
				categoryUrlCache[cat.ID] = ""
			}
		}
		return categoryUrlCache[cat.ID]
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

		breadcrumbs := make([]templates.Breadcrumb, 0, len(lineageBuilder.GetLineage(postResult.Cat.ID)))
		breadcrumbs = append(breadcrumbs, templates.Breadcrumb{
			Name: postResult.Proj.Name,
			Url:  hmnurl.ProjectUrl("/", nil, postResult.Proj.Slug),
		})
		if postResult.Post.CategoryKind == models.CatKindLibraryResource {
			// TODO(asaf): Fetch library root topic for the project and construct breadcrumb for it
		} else {
			lineage := lineageBuilder.GetLineage(postResult.Cat.ID)
			for i, cat := range lineage {
				name := *cat.Name
				if i == 0 {
					switch cat.Kind {
					case models.CatKindForum:
						name = "Forums"
					case models.CatKindBlog:
						name = "Blog"
					}
				}
				breadcrumbs = append(breadcrumbs, templates.Breadcrumb{
					Name: name,
					Url:  getCategoryUrl(postResult.Proj.Subdomain(), cat),
				})
			}
		}

		postItems = append(postItems, templates.PostListItem{
			Title:       postResult.Thread.Title,
			Url:         hmnurl.BuildForumPost(postResult.Proj.Subdomain(), lineageBuilder.GetLineageSlugs(postResult.Cat.ID)[1:], postResult.Post.ID, postResult.Post.ThreadID),
			User:        templates.UserToTemplate(&postResult.User),
			Date:        postResult.Post.PostDate,
			Breadcrumbs: breadcrumbs,
			Unread:      !hasRead,
			Classes:     "post-bg-alternate", // TODO: Should this be the default, and the home page can suppress it?
			Content:     postResult.Post.Preview,
		})
	}
	c.Perf.EndBlock()

	baseData := getBaseData(c)
	baseData.BodyClasses = append(baseData.BodyClasses, "feed")

	var res ResponseData
	res.WriteTemplate("feed.html", FeedData{
		BaseData: baseData,

		Posts:      postItems,
		Pagination: pagination,
	}, c.Perf)

	return res
}

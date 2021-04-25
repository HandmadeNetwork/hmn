package website

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

type FeedData struct {
	templates.BaseData

	Posts      []templates.PostListItem
	Pagination templates.Pagination
}

func Feed(c *RequestContext) ResponseData {
	const postsPerPage = 30

	numPosts, err := db.QueryInt(c.Context(), c.Conn,
		`
		SELECT COUNT(*)
		FROM
			handmade_post AS post
			JOIN handmade_category AS cat ON cat.id = post.category_id
		WHERE
			cat.kind IN ($1, $2, $3, $4)
			AND NOT moderated
		`,
		models.CatTypeForum,
		models.CatTypeBlog,
		models.CatTypeWiki,
		models.CatTypeLibraryResource,
	) // TODO(inarray)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get count of feed posts"))
	}

	numPages := int(math.Ceil(float64(numPosts) / 30))

	page := 1
	pageString := c.PathParams.ByName("page")
	if pageString != "" {
		if pageParsed, err := strconv.Atoi(pageString); err == nil {
			page = pageParsed
		} else {
			return c.Redirect("/feed", http.StatusSeeOther)
		}
	}
	if page < 1 || numPages < page {
		return c.Redirect("/feed", http.StatusSeeOther)
	}

	howManyPostsToSkip := (page - 1) * postsPerPage

	pagination := templates.Pagination{
		Current: page,
		Total:   numPages,

		// TODO: urls
	}

	var currentUserId *int
	if c.CurrentUser != nil {
		currentUserId = &c.CurrentUser.ID
	}

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
			JOIN handmade_category AS cat ON cat.id = thread.category_id
			JOIN handmade_project AS proj ON proj.id = cat.project_id
			LEFT OUTER JOIN handmade_threadlastreadinfo AS tlri ON (
				tlri.thread_id = thread.id
				AND tlri.user_id = $1
			)
			LEFT OUTER JOIN handmade_categorylastreadinfo AS clri ON (
				clri.category_id = cat.id
				AND clri.user_id = $1
			)
			LEFT OUTER JOIN auth_user ON post.author_id = auth_user.id
		WHERE
			cat.kind IN ($2, $3, $4, $5)
			AND post.moderated = FALSE
			AND post.thread_id IS NOT NULL
		ORDER BY postdate DESC
		LIMIT $6 OFFSET $7
		`,
		currentUserId,
		models.CatTypeForum,
		models.CatTypeBlog,
		models.CatTypeWiki,
		models.CatTypeLibraryResource,
		postsPerPage,
		howManyPostsToSkip,
	) // TODO(inarray)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch feed posts"))
	}

	var postItems []templates.PostListItem
	for _, iPostResult := range posts.ToSlice() {
		postResult := iPostResult.(*feedPostQuery)

		hasRead := false
		if postResult.ThreadLastReadTime != nil && postResult.ThreadLastReadTime.After(postResult.Post.PostDate) {
			hasRead = true
		} else if postResult.CatLastReadTime != nil && postResult.CatLastReadTime.After(postResult.Post.PostDate) {
			hasRead = true
		}

		parents := postResult.Cat.GetParents(c.Context(), c.Conn)
		logging.Debug().Interface("parents", parents).Msg("")

		var breadcrumbs []templates.Breadcrumb
		breadcrumbs = append(breadcrumbs, templates.Breadcrumb{
			Name: *postResult.Proj.Name,
			Url:  "nargle", // TODO
		})
		for i := len(parents) - 1; i >= 0; i-- {
			breadcrumbs = append(breadcrumbs, templates.Breadcrumb{
				Name: *parents[i].Name,
				Url:  "nargle", // TODO
			})
		}

		postItems = append(postItems, templates.PostListItem{
			Title:       postResult.Thread.Title,
			Url:         templates.PostUrl(postResult.Post, postResult.Cat.Kind, postResult.Proj.Subdomain()),
			User:        templates.UserToTemplate(&postResult.User),
			Date:        postResult.Post.PostDate,
			Breadcrumbs: breadcrumbs,
			Unread:      !hasRead,
			Classes:     "post-bg-alternate", // TODO: Should this be the default, and the home page can suppress it?
			Content:     postResult.Post.Preview,
		})
	}

	baseData := getBaseData(c)
	baseData.BodyClasses = append(baseData.BodyClasses, "feed")

	var res ResponseData
	res.WriteTemplate("feed.html", FeedData{
		BaseData: baseData,

		Posts:      postItems,
		Pagination: pagination,
	})

	return res
}

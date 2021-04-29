package website

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

func ForumCategory(c *RequestContext) ResponseData {
	const threadsPerPage = 25

	catPath := c.PathParams["cats"]
	catSlugs := strings.Split(catPath, "/")

	catSlug := catSlugs[len(catSlugs)-1]
	if len(catSlugs) == 1 {
		catSlug = ""
	}

	// TODO: Is this query right? Do we need to do a better special case for when it's the root category?
	currentCatId, err := db.QueryInt(c.Context(), c.Conn,
		`
		SELECT id
		FROM handmade_category
		WHERE
			slug = $1
			AND kind = $2
			AND project_id = $3
		`,
		catSlug,
		models.CatKindForum,
		c.CurrentProject.ID,
	)
	if err != nil {
		panic(oops.New(err, "failed to get current category id"))
	}

	numThreads, err := db.QueryInt(c.Context(), c.Conn,
		`
		SELECT COUNT(*)
		FROM handmade_thread AS thread
		WHERE
			thread.category_id = $1
			AND NOT thread.deleted
		`,
		currentCatId,
	)
	if err != nil {
		panic(oops.New(err, "failed to get count of threads"))
	}

	numPages := int(math.Ceil(float64(numThreads) / threadsPerPage))

	page := 1
	pageString, hasPage := c.PathParams["page"]
	if hasPage && pageString != "" {
		if pageParsed, err := strconv.Atoi(pageString); err == nil {
			page = pageParsed
		} else {
			return c.Redirect("/feed", http.StatusSeeOther) // TODO
		}
	}
	if page < 1 || numPages < page {
		return c.Redirect("/feed", http.StatusSeeOther) // TODO
	}

	howManyThreadsToSkip := (page - 1) * threadsPerPage

	var currentUserId *int
	if c.CurrentUser != nil {
		currentUserId = &c.CurrentUser.ID
	}

	type mainPostsQueryResult struct {
		Thread             models.Thread `db:"thread"`
		FirstPost          models.Post   `db:"firstpost"`
		LastPost           models.Post   `db:"lastpost"`
		ThreadLastReadTime *time.Time    `db:"tlri.lastread"`
		CatLastReadTime    *time.Time    `db:"clri.lastread"`
	}
	itMainThreads, err := db.Query(c.Context(), c.Conn, mainPostsQueryResult{},
		`
		SELECT $columns
		FROM
			handmade_thread AS thread
			JOIN handmade_post AS firstpost ON thread.first_id = firstpost.id
			JOIN handmade_post AS lastpost ON thread.last_id = lastpost.id
			LEFT OUTER JOIN handmade_threadlastreadinfo AS tlri ON (
				tlri.thread_id = thread.id
				AND tlri.user_id = $2
			)
			LEFT OUTER JOIN handmade_categorylastreadinfo AS clri ON (
				clri.category_id = $1
				AND clri.user_id = $2
			)
			-- LEFT OUTER JOIN auth_user ON post.author_id = auth_user.id
		WHERE
			thread.category_id = $1
			AND NOT thread.deleted
		ORDER BY lastpost.postdate DESC
		LIMIT $3 OFFSET $4
		`,
		currentCatId,
		currentUserId,
		threadsPerPage,
		howManyThreadsToSkip,
	)
	if err != nil {
		panic(oops.New(err, "failed to fetch threads"))
	}

	var res ResponseData

	for _, irow := range itMainThreads.ToSlice() {
		row := irow.(*mainPostsQueryResult)
		res.Write([]byte(fmt.Sprintf("%s\n", row.Thread.Title)))
	}

	// ---------------------
	// Subcategory things
	// ---------------------

	c.Perf.StartBlock("SQL", "Fetch subcategories")
	type queryResult struct {
		Cat models.Category `db:"cat"`
	}
	itSubcats, err := db.Query(c.Context(), c.Conn, queryResult{},
		`
		WITH current AS (
			SELECT id
			FROM handmade_category
			WHERE
				slug = $1
				AND kind = $2
				AND project_id = $3
		)
		SELECT $columns
		FROM
			handmade_category AS cat,
			current
		WHERE
			cat.id = current.id
			OR cat.parent_id = current.id
		`,
		catSlug,
		models.CatKindForum,
		c.CurrentProject.ID,
	)
	if err != nil {
		panic(oops.New(err, "failed to fetch subcategories"))
	}
	c.Perf.EndBlock()

	_ = itSubcats // TODO: Actually query subcategory post data

	return res
}

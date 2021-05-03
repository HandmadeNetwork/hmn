package website

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/templates"

	"github.com/jackc/pgx/v4/pgxpool"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

type forumCategoryData struct {
	templates.BaseData

	CategoryUrl string
	Threads     []templates.ThreadListItem
	Pagination  templates.Pagination
}

func ForumCategory(c *RequestContext) ResponseData {
	const threadsPerPage = 25

	catPath := c.PathParams["cats"]
	catSlugs := strings.Split(catPath, "/")
	currentCatId := fetchCatIdFromSlugs(c.Context(), c.Conn, catSlugs, c.CurrentProject.ID)
	categoryUrls := GetProjectCategoryUrls(c.Context(), c.Conn, c.CurrentProject.ID)

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
		FirstUser          *models.User  `db:"firstuser"`
		LastUser           *models.User  `db:"lastuser"`
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
			LEFT JOIN auth_user AS firstuser ON firstpost.author_id = firstuser.id
			LEFT JOIN auth_user AS lastuser ON lastpost.author_id = lastuser.id
			LEFT JOIN handmade_threadlastreadinfo AS tlri ON (
				tlri.thread_id = thread.id
				AND tlri.user_id = $2
			)
			LEFT JOIN handmade_categorylastreadinfo AS clri ON (
				clri.category_id = $1
				AND clri.user_id = $2
			)
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
	defer itMainThreads.Close()

	var threads []templates.ThreadListItem
	for _, irow := range itMainThreads.ToSlice() {
		row := irow.(*mainPostsQueryResult)

		hasRead := false
		if row.ThreadLastReadTime != nil && row.ThreadLastReadTime.After(row.LastPost.PostDate) {
			hasRead = true
		} else if row.CatLastReadTime != nil && row.CatLastReadTime.After(row.LastPost.PostDate) {
			hasRead = true
		}

		threads = append(threads, templates.ThreadListItem{
			Title: row.Thread.Title,
			Url:   ThreadUrl(row.Thread, models.CatKindForum, categoryUrls[currentCatId]),

			FirstUser: templates.UserToTemplate(row.FirstUser),
			FirstDate: row.FirstPost.PostDate,
			LastUser:  templates.UserToTemplate(row.LastUser),
			LastDate:  row.LastPost.PostDate,

			Unread: !hasRead,
		})
	}

	// ---------------------
	// Subcategory things
	// ---------------------

	//c.Perf.StartBlock("SQL", "Fetch subcategories")
	//type queryResult struct {
	//	Cat models.Category `db:"cat"`
	//}
	//itSubcats, err := db.Query(c.Context(), c.Conn, queryResult{},
	//	`
	//	WITH current AS (
	//		SELECT id
	//		FROM handmade_category
	//		WHERE
	//			slug = $1
	//			AND kind = $2
	//			AND project_id = $3
	//	)
	//	SELECT $columns
	//	FROM
	//		handmade_category AS cat,
	//		current
	//	WHERE
	//		cat.id = current.id
	//		OR cat.parent_id = current.id
	//	`,
	//	catSlug,
	//	models.CatKindForum,
	//	c.CurrentProject.ID,
	//)
	//if err != nil {
	//	panic(oops.New(err, "failed to fetch subcategories"))
	//}
	//c.Perf.EndBlock()

	//_ = itSubcats // TODO: Actually query subcategory post data

	baseData := getBaseData(c)
	baseData.Title = *c.CurrentProject.Name + " Forums"
	baseData.Breadcrumbs = []templates.Breadcrumb{
		{
			Name: *c.CurrentProject.Name,
			Url:  hmnurl.ProjectUrl("/", nil, c.CurrentProject.Subdomain()),
		},
		{
			Name:    "Forums",
			Url:     categoryUrls[currentCatId],
			Current: true,
		},
	}

	var res ResponseData
	err = res.WriteTemplate("forum_category.html", forumCategoryData{
		BaseData:    baseData,
		CategoryUrl: categoryUrls[currentCatId],
		Threads:     threads,
		Pagination: templates.Pagination{
			Current: page,
			Total:   numPages,

			FirstUrl:    categoryUrls[currentCatId],
			LastUrl:     fmt.Sprintf("%s/%d", categoryUrls[currentCatId], numPages),
			NextUrl:     fmt.Sprintf("%s/%d", categoryUrls[currentCatId], page+1),
			PreviousUrl: fmt.Sprintf("%s/%d", categoryUrls[currentCatId], page-1),
		},
	}, c.Perf)
	if err != nil {
		panic(err)
	}

	return res
}

func fetchCatIdFromSlugs(ctx context.Context, conn *pgxpool.Pool, catSlugs []string, projectId int) int {
	if len(catSlugs) == 1 {
		var err error
		currentCatId, err := db.QueryInt(ctx, conn,
			`
			SELECT cat.id
			FROM
				handmade_category AS cat
				JOIN handmade_project AS proj ON proj.forum_id = cat.id
			WHERE
				proj.id = $1
				AND cat.kind = $2
			`,
			projectId,
			models.CatKindForum,
		)
		if err != nil {
			panic(oops.New(err, "failed to get root category id"))
		}

		return currentCatId
	} else {
		var err error
		currentCatId, err := db.QueryInt(ctx, conn,
			`
			SELECT id
			FROM handmade_category
			WHERE
				slug = $1
				AND kind = $2
				AND project_id = $3
			`,
			catSlugs[len(catSlugs)-1],
			models.CatKindForum,
			projectId,
		)
		if err != nil {
			panic(oops.New(err, "failed to get current category id"))
		}

		return currentCatId
	}
}

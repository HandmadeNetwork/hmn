package website

import (
	"errors"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

type forumCategoryData struct {
	templates.BaseData

	NewThreadUrl  string
	MarkReadUrl   string
	Threads       []templates.ThreadListItem
	Pagination    templates.Pagination
	Subcategories []forumSubcategoryData
}

type forumSubcategoryData struct {
	Name         string
	Url          string
	Threads      []templates.ThreadListItem
	TotalThreads int
}

func ForumCategory(c *RequestContext) ResponseData {
	const threadsPerPage = 25

	c.Perf.StartBlock("SQL", "Fetch category tree")
	categoryTree := models.GetFullCategoryTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeCategoryLineageBuilder(categoryTree)
	c.Perf.EndBlock()

	currentCatId, valid := validateSubforums(lineageBuilder, c.CurrentProject, c.PathParams["cats"])
	if !valid {
		return FourOhFour(c)
	}

	currentSubforumSlugs := lineageBuilder.GetSubforumLineageSlugs(currentCatId)

	c.Perf.StartBlock("SQL", "Fetch count of page threads")
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
	c.Perf.EndBlock()

	numPages := int(math.Ceil(float64(numThreads) / threadsPerPage))

	page := 1
	pageString, hasPage := c.PathParams["page"]
	if hasPage && pageString != "" {
		if pageParsed, err := strconv.Atoi(pageString); err == nil {
			page = pageParsed
		} else {
			return c.Redirect(hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs, 1), http.StatusSeeOther)
		}
	}
	if page < 1 || numPages < page {
		return c.Redirect(hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs, utils.IntClamp(1, page, numPages)), http.StatusSeeOther)
	}

	howManyThreadsToSkip := (page - 1) * threadsPerPage

	var currentUserId *int
	if c.CurrentUser != nil {
		currentUserId = &c.CurrentUser.ID
	}

	c.Perf.StartBlock("SQL", "Fetch page threads")
	type threadQueryResult struct {
		Thread             models.Thread `db:"thread"`
		FirstPost          models.Post   `db:"firstpost"`
		LastPost           models.Post   `db:"lastpost"`
		FirstUser          *models.User  `db:"firstuser"`
		LastUser           *models.User  `db:"lastuser"`
		ThreadLastReadTime *time.Time    `db:"tlri.lastread"`
		CatLastReadTime    *time.Time    `db:"clri.lastread"`
	}
	itMainThreads, err := db.Query(c.Context(), c.Conn, threadQueryResult{},
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
	c.Perf.EndBlock()
	defer itMainThreads.Close()

	makeThreadListItem := func(row *threadQueryResult) templates.ThreadListItem {
		hasRead := false
		if row.ThreadLastReadTime != nil && row.ThreadLastReadTime.After(row.LastPost.PostDate) {
			hasRead = true
		} else if row.CatLastReadTime != nil && row.CatLastReadTime.After(row.LastPost.PostDate) {
			hasRead = true
		}

		return templates.ThreadListItem{
			Title:     row.Thread.Title,
			Url:       hmnurl.BuildForumThread(c.CurrentProject.Slug, lineageBuilder.GetSubforumLineageSlugs(row.Thread.CategoryID), row.Thread.ID, row.Thread.Title, 1),
			FirstUser: templates.UserToTemplate(row.FirstUser, c.Theme),
			FirstDate: row.FirstPost.PostDate,
			LastUser:  templates.UserToTemplate(row.LastUser, c.Theme),
			LastDate:  row.LastPost.PostDate,

			Unread: !hasRead,
		}
	}

	var threads []templates.ThreadListItem
	for _, irow := range itMainThreads.ToSlice() {
		row := irow.(*threadQueryResult)
		threads = append(threads, makeThreadListItem(row))
	}

	// ---------------------
	// Subcategory things
	// ---------------------

	var subcats []forumSubcategoryData
	if page == 1 {
		subcatNodes := categoryTree[currentCatId].Children

		for _, catNode := range subcatNodes {
			c.Perf.StartBlock("SQL", "Fetch count of subcategory threads")
			// TODO(asaf): [PERF] [MINOR] Consider replacing querying count per subcat with a single query for all cats with GROUP BY.
			numThreads, err := db.QueryInt(c.Context(), c.Conn,
				`
				SELECT COUNT(*)
				FROM handmade_thread AS thread
				WHERE
					thread.category_id = $1
					AND NOT thread.deleted
				`,
				catNode.ID,
			)
			if err != nil {
				panic(oops.New(err, "failed to get count of threads"))
			}
			c.Perf.EndBlock()

			c.Perf.StartBlock("SQL", "Fetch subcategory threads")
			// TODO(asaf): [PERF] [MINOR] Consider batching these.
			itThreads, err := db.Query(c.Context(), c.Conn, threadQueryResult{},
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
				LIMIT 3
				`,
				catNode.ID,
				currentUserId,
			)
			if err != nil {
				panic(err)
			}
			defer itThreads.Close()
			c.Perf.EndBlock()

			var threads []templates.ThreadListItem
			for _, irow := range itThreads.ToSlice() {
				threadRow := irow.(*threadQueryResult)
				threads = append(threads, makeThreadListItem(threadRow))
			}

			subcats = append(subcats, forumSubcategoryData{
				Name:         *catNode.Name,
				Url:          hmnurl.BuildForumCategory(c.CurrentProject.Slug, lineageBuilder.GetSubforumLineageSlugs(catNode.ID), 1),
				Threads:      threads,
				TotalThreads: numThreads,
			})
		}
	}

	// ---------------------
	// Template assembly
	// ---------------------

	baseData := getBaseData(c)
	baseData.Title = c.CurrentProject.Name + " Forums"
	baseData.Breadcrumbs = []templates.Breadcrumb{ // TODO(ben): This is wrong; it needs to account for subcategories.
		{
			Name: c.CurrentProject.Name,
			Url:  hmnurl.BuildProjectHomepage(c.CurrentProject.Slug),
		},
		{
			Name:    "Forums",
			Url:     hmnurl.BuildForumCategory(c.CurrentProject.Slug, nil, 1),
			Current: true,
		},
	}

	currentSubforums := lineageBuilder.GetSubforumLineage(currentCatId)
	for i, subforum := range currentSubforums {
		baseData.Breadcrumbs = append(baseData.Breadcrumbs, templates.Breadcrumb{
			Name: *subforum.Name, // NOTE(asaf): All subforum categories must have names.
			Url:  hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs[0:i+1], 1),
		})
	}

	var res ResponseData
	err = res.WriteTemplate("forum_category.html", forumCategoryData{
		BaseData:     baseData,
		NewThreadUrl: hmnurl.BuildForumNewThread(c.CurrentProject.Slug, currentSubforumSlugs, false),
		MarkReadUrl:  hmnurl.BuildMarkRead(currentCatId),
		Threads:      threads,
		Pagination: templates.Pagination{
			Current: page,
			Total:   numPages,

			FirstUrl:    hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs, 1),
			LastUrl:     hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs, numPages),
			NextUrl:     hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs, utils.IntClamp(1, page+1, numPages)),
			PreviousUrl: hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs, utils.IntClamp(1, page-1, numPages)),
		},
		Subcategories: subcats,
	}, c.Perf)
	if err != nil {
		panic(err)
	}

	return res
}

type forumThreadData struct {
	templates.BaseData

	Thread templates.Thread
	Posts  []templates.Post

	CategoryUrl string
	ReplyUrl    string
	Pagination  templates.Pagination
}

var threadViewPostsPerPage = 15

func ForumThread(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch category tree")
	categoryTree := models.GetFullCategoryTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeCategoryLineageBuilder(categoryTree)
	c.Perf.EndBlock()

	currentCatId, valid := validateSubforums(lineageBuilder, c.CurrentProject, c.PathParams["cats"])
	if !valid {
		return FourOhFour(c)
	}

	threadId, err := strconv.Atoi(c.PathParams["threadid"])
	if err != nil {
		return FourOhFour(c)
	}

	currentSubforumSlugs := lineageBuilder.GetSubforumLineageSlugs(currentCatId)

	c.Perf.StartBlock("SQL", "Fetch current thread")
	type threadQueryResult struct {
		Thread models.Thread `db:"thread"`
	}
	irow, err := db.QueryOne(c.Context(), c.Conn, threadQueryResult{},
		`
		SELECT $columns
		FROM
			handmade_thread AS thread
			JOIN handmade_category AS cat ON cat.id = thread.category_id
		WHERE
			thread.id = $1
			AND NOT thread.deleted
			AND cat.id = $2
		`,
		threadId,
		currentCatId, // NOTE(asaf): This verifies that the requested thread is under the requested subforum.
	)
	c.Perf.EndBlock()
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return FourOhFour(c)
		} else {
			panic(err)
		}
	}
	thread := irow.(*threadQueryResult).Thread

	numPosts, err := db.QueryInt(c.Context(), c.Conn,
		`
		SELECT COUNT(*)
		FROM handmade_post
		WHERE
			thread_id = $1
			AND NOT deleted
		`,
		thread.ID,
	)
	if err != nil {
		panic(oops.New(err, "failed to get count of posts for thread"))
	}
	page, numPages, ok := getPageInfo(c.PathParams["page"], numPosts, threadViewPostsPerPage)
	if !ok {
		urlNoPage := hmnurl.BuildForumThread(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, thread.Title, 1)
		return c.Redirect(urlNoPage, http.StatusSeeOther)
	}
	pagination := templates.Pagination{
		Current: page,
		Total:   numPages,

		FirstUrl:    hmnurl.BuildForumThread(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, thread.Title, 1),
		LastUrl:     hmnurl.BuildForumThread(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, thread.Title, numPages),
		NextUrl:     hmnurl.BuildForumThread(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, thread.Title, utils.IntClamp(1, page+1, numPages)),
		PreviousUrl: hmnurl.BuildForumThread(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, thread.Title, utils.IntClamp(1, page-1, numPages)),
	}

	c.Perf.StartBlock("SQL", "Fetch posts")
	type postsQueryResult struct {
		Post   models.Post        `db:"post"`
		Ver    models.PostVersion `db:"ver"`
		Author *models.User       `db:"author"`
		Editor *models.User       `db:"editor"`
	}
	itPosts, err := db.Query(c.Context(), c.Conn, postsQueryResult{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			JOIN handmade_postversion AS ver ON post.current_id = ver.id
			LEFT JOIN auth_user AS author ON post.author_id = author.id
			LEFT JOIN auth_user AS editor ON ver.editor_id = editor.id
		WHERE
			post.thread_id = $1
			AND NOT post.deleted
		ORDER BY postdate
		LIMIT $2 OFFSET $3
		`,
		thread.ID,
		threadViewPostsPerPage,
		(page-1)*threadViewPostsPerPage,
	)
	c.Perf.EndBlock()
	if err != nil {
		panic(err)
	}
	defer itPosts.Close()

	var posts []templates.Post
	for _, irow := range itPosts.ToSlice() {
		row := irow.(*postsQueryResult)

		post := templates.PostToTemplate(&row.Post, row.Author, c.Theme)
		post.AddContentVersion(row.Ver, row.Editor, c.Theme)
		post.AddUrls(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, post.ID)

		posts = append(posts, post)
	}

	baseData := getBaseData(c)
	baseData.Title = thread.Title
	// TODO(asaf): Set breadcrumbs

	var res ResponseData
	err = res.WriteTemplate("forum_thread.html", forumThreadData{
		BaseData:    baseData,
		Thread:      templates.ThreadToTemplate(&thread),
		Posts:       posts,
		CategoryUrl: hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs, 1),
		ReplyUrl:    hmnurl.BuildForumPostReply(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, *thread.FirstID),
		Pagination:  pagination,
	}, c.Perf)
	if err != nil {
		panic(err)
	}

	return res
}

func ForumPostRedirect(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch category tree")
	categoryTree := models.GetFullCategoryTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeCategoryLineageBuilder(categoryTree)
	c.Perf.EndBlock()

	currentCatId, valid := validateSubforums(lineageBuilder, c.CurrentProject, c.PathParams["cats"])
	if !valid {
		return FourOhFour(c)
	}

	requestedThreadId, err := strconv.Atoi(c.PathParams["threadid"])
	if err != nil {
		return FourOhFour(c)
	}

	requestedPostId, err := strconv.Atoi(c.PathParams["postid"])
	if err != nil {
		return FourOhFour(c)
	}

	c.Perf.StartBlock("SQL", "Fetch post ids for thread")
	type postQuery struct {
		PostID int `db:"post.id"`
	}
	postQueryResult, err := db.Query(c.Context(), c.Conn, postQuery{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
		WHERE
			post.category_id = $1
			AND post.thread_id = $2
			AND NOT post.deleted
		ORDER BY postdate
		`,
		currentCatId,
		requestedThreadId,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch post ids"))
	}
	postQuerySlice := postQueryResult.ToSlice()
	c.Perf.EndBlock()
	postIdx := -1
	for i, id := range postQuerySlice {
		if id.(*postQuery).PostID == requestedPostId {
			postIdx = i
			break
		}
	}
	if postIdx == -1 {
		return FourOhFour(c)
	}

	c.Perf.StartBlock("SQL", "Fetch thread title")
	type threadTitleQuery struct {
		ThreadTitle string `db:"thread.title"`
	}
	threadTitleQueryResult, err := db.QueryOne(c.Context(), c.Conn, threadTitleQuery{},
		`
		SELECT $columns
		FROM handmade_thread AS thread
		WHERE thread.id = $1
		`,
		requestedThreadId,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch thread title"))
	}
	c.Perf.EndBlock()
	threadTitle := threadTitleQueryResult.(*threadTitleQuery).ThreadTitle

	page := (postIdx / threadViewPostsPerPage) + 1

	return c.Redirect(hmnurl.BuildForumThreadWithPostHash(
		c.CurrentProject.Slug,
		lineageBuilder.GetSubforumLineageSlugs(currentCatId),
		requestedThreadId,
		threadTitle,
		page,
		requestedPostId,
	), http.StatusSeeOther)
}

type editorData struct {
	templates.BaseData
	SubmitUrl    string
	PostTitle    string
	PostBody     string
	SubmitLabel  string
	PreviewLabel string
}

func ForumNewThread(c *RequestContext) ResponseData {
	if c.Req.Method == http.MethodPost {
		// TODO: Get preview data
	}

	baseData := getBaseData(c)
	baseData.Title = "Create New Thread"
	baseData.MathjaxEnabled = true
	// TODO(ben): Set breadcrumbs

	c.Perf.StartBlock("SQL", "Fetch category tree")
	categoryTree := models.GetFullCategoryTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeCategoryLineageBuilder(categoryTree)
	c.Perf.EndBlock()

	currentCatId, valid := validateSubforums(lineageBuilder, c.CurrentProject, c.PathParams["cats"])
	if !valid {
		return FourOhFour(c)
	}

	var res ResponseData
	err := res.WriteTemplate("editor.html", editorData{
		BaseData:     baseData,
		SubmitUrl:    hmnurl.BuildForumNewThread(c.CurrentProject.Slug, lineageBuilder.GetSubforumLineageSlugs(currentCatId), true),
		SubmitLabel:  "Post New Thread",
		PreviewLabel: "Preview",
	}, c.Perf)
	// err := res.WriteTemplate("forum_thread.html", forumThreadData{
	// 	BaseData:    baseData,
	// 	Thread:      templates.ThreadToTemplate(&thread),
	// 	Posts:       posts,
	// 	CategoryUrl: hmnurl.BuildForumCategory(c.CurrentProject.Slug, currentSubforumSlugs, 1),
	// 	ReplyUrl:    hmnurl.BuildForumPostReply(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, *thread.FirstID),
	// 	Pagination:  pagination,
	// }, c.Perf)
	if err != nil {
		panic(err)
	}

	return res
}

func ForumNewThreadSubmit(c *RequestContext) ResponseData {
	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}

	c.Perf.StartBlock("SQL", "Fetch category tree")
	categoryTree := models.GetFullCategoryTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeCategoryLineageBuilder(categoryTree)
	c.Perf.EndBlock()

	currentCatId, valid := validateSubforums(lineageBuilder, c.CurrentProject, c.PathParams["cats"])
	if !valid {
		return FourOhFour(c)
	}

	c.Req.ParseForm()

	title := c.Req.Form.Get("title")
	unparsed := c.Req.Form.Get("body")
	sticky := false
	if c.CurrentUser.IsStaff && c.Req.Form.Get("sticky") != "" {
		sticky = true
	}

	parsed := parsing.ParsePostInput(unparsed, false)

	// Create thread
	var threadId int
	err = tx.QueryRow(c.Context(),
		`
		INSERT INTO handmade_thread (title, sticky, locked, category_id)
		RETURNING id
		`,
		title,
		sticky,
		false,
		currentCatId,
	).Scan(&threadId)
	if err != nil {
		panic(oops.New(err, "failed to create thread"))
	}

	// Create post version
	_, err = tx.Exec(c.Context(),
		`
		INSERT INTO handmade_postversion (post_id, text_raw, text_parsed)
		VALUES ($1, $2, $3)
		`,
		// TODO: post id
		unparsed,
		parsed,
	)

	err = tx.Commit(c.Context())
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create new forum thread"))
	}

	// TODO: Redirect to newly created thread
	return c.Redirect(hmnurl.BuildForumNewThread(models.HMNProjectSlug, nil, false), http.StatusSeeOther)
}

func validateSubforums(lineageBuilder *models.CategoryLineageBuilder, project *models.Project, catPath string) (int, bool) {
	if project.ForumID == nil {
		return -1, false
	}

	subforumCatId := *project.ForumID
	if len(catPath) == 0 {
		return subforumCatId, true
	}

	catPath = strings.ToLower(catPath)
	valid := false
	catSlugs := strings.Split(catPath, "/")
	lastSlug := catSlugs[len(catSlugs)-1]
	if len(lastSlug) > 0 {
		lastSlugCatId := lineageBuilder.FindIdBySlug(project.ID, lastSlug)
		if lastSlugCatId != -1 {
			subforumSlugs := lineageBuilder.GetSubforumLineageSlugs(lastSlugCatId)
			allMatch := true
			for i, subforum := range subforumSlugs {
				if subforum != catSlugs[i] {
					allMatch = false
					break
				}
			}
			valid = allMatch
		}
		if valid {
			subforumCatId = lastSlugCatId
		}
	}
	return subforumCatId, valid
}

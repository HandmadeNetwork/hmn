package website

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

type forumData struct {
	templates.BaseData

	NewThreadUrl string
	MarkReadUrl  string
	Threads      []templates.ThreadListItem
	Pagination   templates.Pagination
	Subforums    []forumSubforumData
}

type forumSubforumData struct {
	Name         string
	Url          string
	Threads      []templates.ThreadListItem
	TotalThreads int
}

type editActionType string

type editorData struct {
	templates.BaseData
	SubmitUrl   string
	SubmitLabel string

	// The following are filled out automatically by the
	// getEditorDataFor* functions.
	Title               string
	CanEditTitle        bool
	IsEditing           bool
	EditInitialContents string
	PostReplyingTo      *templates.Post
}

func getEditorDataForNew(baseData templates.BaseData, replyPost *templates.Post) editorData {
	result := editorData{
		BaseData:       baseData,
		CanEditTitle:   replyPost == nil,
		PostReplyingTo: replyPost,
	}

	if replyPost != nil {
		result.Title = "Replying to post"
	}

	return result
}

func getEditorDataForEdit(baseData templates.BaseData, p postAndRelatedModels) editorData {
	return editorData{
		BaseData:            baseData,
		Title:               p.Thread.Title,
		CanEditTitle:        p.Thread.FirstID == p.Post.ID,
		IsEditing:           true,
		EditInitialContents: p.CurrentVersion.TextRaw,
	}
}

func Forum(c *RequestContext) ResponseData {
	const threadsPerPage = 25

	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	currentSubforumSlugs := cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID)

	c.Perf.StartBlock("SQL", "Fetch count of page threads")
	numThreads, err := db.QueryInt(c.Context(), c.Conn,
		`
		SELECT COUNT(*)
		FROM handmade_thread AS thread
		WHERE
			thread.subforum_id = $1
			AND NOT thread.deleted
		`,
		cd.SubforumID,
	)
	if err != nil {
		panic(oops.New(err, "failed to get count of threads"))
	}
	c.Perf.EndBlock()

	numPages := utils.NumPages(numThreads, threadsPerPage)
	page, ok := ParsePageNumber(c, "page", numPages)
	if !ok {
		c.Redirect(hmnurl.BuildForum(c.CurrentProject.Slug, currentSubforumSlugs, page), http.StatusSeeOther)
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
		ForumLastReadTime  *time.Time    `db:"slri.lastread"`
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
			LEFT JOIN handmade_subforumlastreadinfo AS slri ON (
				slri.subforum_id = $1
				AND slri.user_id = $2
			)
		WHERE
			thread.subforum_id = $1
			AND NOT thread.deleted
		ORDER BY lastpost.postdate DESC
		LIMIT $3 OFFSET $4
		`,
		cd.SubforumID,
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
		} else if row.ForumLastReadTime != nil && row.ForumLastReadTime.After(row.LastPost.PostDate) {
			hasRead = true
		}

		return templates.ThreadListItem{
			Title:     row.Thread.Title,
			Url:       hmnurl.BuildForumThread(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(*row.Thread.SubforumID), row.Thread.ID, row.Thread.Title, 1),
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
	// Subforum things
	// ---------------------

	var subforums []forumSubforumData
	if page == 1 {
		subforumNodes := cd.SubforumTree[cd.SubforumID].Children

		for _, sfNode := range subforumNodes {
			c.Perf.StartBlock("SQL", "Fetch count of subforum threads")
			numThreads, err := db.QueryInt(c.Context(), c.Conn,
				`
				SELECT COUNT(*)
				FROM handmade_thread AS thread
				WHERE
					thread.subforum_id = $1
					AND NOT thread.deleted
				`,
				sfNode.ID,
			)
			if err != nil {
				panic(oops.New(err, "failed to get count of threads"))
			}
			c.Perf.EndBlock()

			c.Perf.StartBlock("SQL", "Fetch subforum threads")
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
					LEFT JOIN handmade_subforumlastreadinfo AS slri ON (
						slri.subforum_id = $1
						AND slri.user_id = $2
					)
				WHERE
					thread.subforum_id = $1
					AND NOT thread.deleted
				ORDER BY lastpost.postdate DESC
				LIMIT 3
				`,
				sfNode.ID,
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

			subforums = append(subforums, forumSubforumData{
				Name:         sfNode.Name,
				Url:          hmnurl.BuildForum(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(sfNode.ID), 1),
				Threads:      threads,
				TotalThreads: numThreads,
			})
		}
	}

	// ---------------------
	// Template assembly
	// ---------------------

	baseData := getBaseData(
		c,
		fmt.Sprintf("%s Forums", c.CurrentProject.Name),
		SubforumBreadcrumbs(cd.LineageBuilder, c.CurrentProject, cd.SubforumID),
	)

	var res ResponseData
	res.MustWriteTemplate("forum.html", forumData{
		BaseData:     baseData,
		NewThreadUrl: hmnurl.BuildForumNewThread(c.CurrentProject.Slug, currentSubforumSlugs, false),
		MarkReadUrl:  hmnurl.BuildForumMarkRead(c.CurrentProject.Slug, cd.SubforumID),
		Threads:      threads,
		Pagination: templates.Pagination{
			Current: page,
			Total:   numPages,

			FirstUrl:    hmnurl.BuildForum(c.CurrentProject.Slug, currentSubforumSlugs, 1),
			LastUrl:     hmnurl.BuildForum(c.CurrentProject.Slug, currentSubforumSlugs, numPages),
			NextUrl:     hmnurl.BuildForum(c.CurrentProject.Slug, currentSubforumSlugs, utils.IntClamp(1, page+1, numPages)),
			PreviousUrl: hmnurl.BuildForum(c.CurrentProject.Slug, currentSubforumSlugs, utils.IntClamp(1, page-1, numPages)),
		},
		Subforums: subforums,
	}, c.Perf)
	return res
}

func ForumMarkRead(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	sfId, err := strconv.Atoi(c.PathParams["sfid"])
	if err != nil {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	sfIds := []int{sfId}
	if sfId == 0 {
		// Mark literally everything as read
		_, err := tx.Exec(c.Context(),
			`
			UPDATE auth_user
			SET marked_all_read_at = NOW()
			WHERE id = $1
			`,
			c.CurrentUser.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to mark all posts as read"))
		}

		// Delete thread unread info
		_, err = tx.Exec(c.Context(),
			`
			DELETE FROM handmade_threadlastreadinfo
			WHERE user_id = $1;
			`,
			c.CurrentUser.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete thread unread info"))
		}

		// Delete subforum unread info
		_, err = tx.Exec(c.Context(),
			`
			DELETE FROM handmade_subforumlastreadinfo
			WHERE user_id = $1;
			`,
			c.CurrentUser.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete subforum unread info"))
		}
	} else {
		c.Perf.StartBlock("SQL", "Update SLRIs")
		_, err = tx.Exec(c.Context(),
			`
		INSERT INTO handmade_subforumlastreadinfo (subforum_id, user_id, lastread)
			SELECT id, $2, $3
			FROM handmade_subforum
			WHERE id = ANY ($1)
		ON CONFLICT (subforum_id, user_id) DO UPDATE
			SET lastread = EXCLUDED.lastread
		`,
			sfIds,
			c.CurrentUser.ID,
			time.Now(),
		)
		c.Perf.EndBlock()
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update forum slris"))
		}

		c.Perf.StartBlock("SQL", "Delete TLRIs")
		_, err = tx.Exec(c.Context(),
			`
		DELETE FROM handmade_threadlastreadinfo
		WHERE
			user_id = $2
			AND thread_id IN (
				SELECT id
				FROM handmade_thread
				WHERE
					subforum_id = ANY ($1)
			)
		`,
			sfIds,
			c.CurrentUser.ID,
		)
		c.Perf.EndBlock()
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete unnecessary tlris"))
		}
	}

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit SLRI/TLRI updates"))
	}

	var redirUrl string
	if sfId == 0 {
		redirUrl = hmnurl.BuildFeed()
	} else {
		redirUrl = hmnurl.BuildForum(c.CurrentProject.Slug, lineageBuilder.GetSubforumLineageSlugs(sfId), 1)
	}
	return c.Redirect(redirUrl, http.StatusSeeOther)
}

type forumThreadData struct {
	templates.BaseData

	Thread templates.Thread
	Posts  []templates.Post

	SubforumUrl string
	ReplyUrl    string
	Pagination  templates.Pagination
}

var threadViewPostsPerPage = 15

func ForumThread(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	currentSubforumSlugs := cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID)

	thread := FetchThread(c.Context(), c.Conn, cd.ThreadID)

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
	_, postsAndStuff := FetchThreadPostsAndStuff(
		c.Context(),
		c.Conn,
		cd.ThreadID,
		page, threadViewPostsPerPage,
	)
	c.Perf.EndBlock()

	c.Perf.StartBlock("TEMPLATE", "Create template posts")
	var posts []templates.Post
	for _, p := range postsAndStuff {
		post := templates.PostToTemplate(&p.Post, p.Author, c.Theme)
		post.AddContentVersion(p.CurrentVersion, p.Editor)
		addForumUrlsToPost(&post, c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, post.ID)

		if p.ReplyPost != nil {
			reply := templates.PostToTemplate(p.ReplyPost, p.ReplyAuthor, c.Theme)
			addForumUrlsToPost(&reply, c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, post.ID)
			post.ReplyPost = &reply
		}

		addAuthorCountsToPost(c.Context(), c.Conn, &post)

		posts = append(posts, post)
	}
	c.Perf.EndBlock()

	// Update thread last read info
	if c.CurrentUser != nil {
		c.Perf.StartBlock("SQL", "Update TLRI")
		_, err = c.Conn.Exec(c.Context(),
			`
		INSERT INTO handmade_threadlastreadinfo (thread_id, user_id, lastread)
		VALUES ($1, $2, $3)
		ON CONFLICT (thread_id, user_id) DO UPDATE
			SET lastread = EXCLUDED.lastread
		`,
			cd.ThreadID,
			c.CurrentUser.ID,
			time.Now(),
		)
		c.Perf.EndBlock()
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update forum tlri"))
		}
	}

	baseData := getBaseData(c, thread.Title, SubforumBreadcrumbs(cd.LineageBuilder, c.CurrentProject, cd.SubforumID))

	var res ResponseData
	res.MustWriteTemplate("forum_thread.html", forumThreadData{
		BaseData:    baseData,
		Thread:      templates.ThreadToTemplate(&thread),
		Posts:       posts,
		SubforumUrl: hmnurl.BuildForum(c.CurrentProject.Slug, currentSubforumSlugs, 1),
		ReplyUrl:    hmnurl.BuildForumPostReply(c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, thread.FirstID),
		Pagination:  pagination,
	}, c.Perf)
	return res
}

func ForumPostRedirect(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
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
			post.thread_id = $1
			AND NOT post.deleted
		ORDER BY postdate
		`,
		cd.ThreadID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch post ids"))
	}
	postQuerySlice := postQueryResult.ToSlice()
	c.Perf.EndBlock()
	postIdx := -1
	for i, id := range postQuerySlice {
		if id.(*postQuery).PostID == cd.PostID {
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
		cd.ThreadID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch thread title"))
	}
	c.Perf.EndBlock()
	threadTitle := threadTitleQueryResult.(*threadTitleQuery).ThreadTitle

	page := (postIdx / threadViewPostsPerPage) + 1

	return c.Redirect(hmnurl.BuildForumThreadWithPostHash(
		c.CurrentProject.Slug,
		cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID),
		cd.ThreadID,
		threadTitle,
		page,
		cd.PostID,
	), http.StatusSeeOther)
}

func ForumNewThread(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	baseData := getBaseData(c, "Create New Thread", SubforumBreadcrumbs(cd.LineageBuilder, c.CurrentProject, cd.SubforumID))
	editData := getEditorDataForNew(baseData, nil)
	editData.SubmitUrl = hmnurl.BuildForumNewThread(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), true)
	editData.SubmitLabel = "Post New Thread"

	var res ResponseData
	res.MustWriteTemplate("editor.html", editData, c.Perf)
	return res
}

func ForumNewThreadSubmit(c *RequestContext) ResponseData {
	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	err = c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "the form data was invalid"))
	}
	title := c.Req.Form.Get("title")
	unparsed := c.Req.Form.Get("body")
	sticky := false
	if c.CurrentUser.IsStaff && c.Req.Form.Get("sticky") != "" {
		sticky = true
	}
	if title == "" {
		return RejectRequest(c, "You must provide a title for your post.")
	}
	if unparsed == "" {
		return RejectRequest(c, "You must provide a body for your post.")
	}

	// Create thread
	var threadId int
	err = tx.QueryRow(c.Context(),
		`
		INSERT INTO handmade_thread (title, sticky, type, project_id, subforum_id, first_id, last_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
		`,
		title,
		sticky,
		models.ThreadTypeForumPost,
		c.CurrentProject.ID,
		cd.SubforumID,
		-1,
		-1,
	).Scan(&threadId)
	if err != nil {
		panic(oops.New(err, "failed to create thread"))
	}

	// Create everything else
	CreateNewPost(c.Context(), tx, c.CurrentProject.ID, threadId, models.ThreadTypeForumPost, c.CurrentUser.ID, nil, unparsed, c.Req.Host)

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create new forum thread"))
	}

	newThreadUrl := hmnurl.BuildForumThread(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), threadId, title, 1)
	return c.Redirect(newThreadUrl, http.StatusSeeOther)
}

func ForumPostReply(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	postData := FetchPostAndStuff(c.Context(), c.Conn, cd.ThreadID, cd.PostID)

	baseData := getBaseData(
		c,
		fmt.Sprintf("Replying to post | %s", cd.SubforumTree[cd.SubforumID].Name),
		ForumThreadBreadcrumbs(cd.LineageBuilder, c.CurrentProject, &postData.Thread),
	)

	replyPost := templates.PostToTemplate(&postData.Post, postData.Author, c.Theme)
	replyPost.AddContentVersion(postData.CurrentVersion, postData.Editor)

	editData := getEditorDataForNew(baseData, &replyPost)
	editData.SubmitUrl = hmnurl.BuildForumPostReply(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), cd.ThreadID, cd.PostID)
	editData.SubmitLabel = "Submit Reply"

	var res ResponseData
	res.MustWriteTemplate("editor.html", editData, c.Perf)
	return res
}

func ForumPostReplySubmit(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	err = c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, oops.New(nil, "the form data was invalid"))
	}
	unparsed := c.Req.Form.Get("body")
	if unparsed == "" {
		return RejectRequest(c, "Your reply cannot be empty.")
	}

	newPostId, _ := CreateNewPost(c.Context(), tx, c.CurrentProject.ID, cd.ThreadID, models.ThreadTypeForumPost, c.CurrentUser.ID, &cd.PostID, unparsed, c.Req.Host)

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to reply to forum post"))
	}

	newPostUrl := hmnurl.BuildForumPost(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), cd.ThreadID, newPostId)
	return c.Redirect(newPostUrl, http.StatusSeeOther)
}

func ForumPostEdit(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	postData := FetchPostAndStuff(c.Context(), c.Conn, cd.ThreadID, cd.PostID)

	title := ""
	if postData.Thread.FirstID == postData.Post.ID {
		title = fmt.Sprintf("Editing \"%s\" | %s", postData.Thread.Title, cd.SubforumTree[cd.SubforumID].Name)
	} else {
		title = fmt.Sprintf("Editing Post | %s", cd.SubforumTree[cd.SubforumID].Name)
	}
	baseData := getBaseData(c, title, ForumThreadBreadcrumbs(cd.LineageBuilder, c.CurrentProject, &postData.Thread))

	editData := getEditorDataForEdit(baseData, postData)
	editData.SubmitUrl = hmnurl.BuildForumPostEdit(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), cd.ThreadID, cd.PostID)
	editData.SubmitLabel = "Submit Edited Post"

	var res ResponseData
	res.MustWriteTemplate("editor.html", editData, c.Perf)
	return res
}

func ForumPostEditSubmit(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	c.Req.ParseForm()
	unparsed := c.Req.Form.Get("body")
	editReason := c.Req.Form.Get("editreason")
	if unparsed == "" {
		return RejectRequest(c, "You must provide a body for your post.")
	}

	CreatePostVersion(c.Context(), tx, cd.PostID, unparsed, c.Req.Host, editReason, &c.CurrentUser.ID)

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to edit forum post"))
	}

	postUrl := hmnurl.BuildForumPost(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), cd.ThreadID, cd.PostID)
	return c.Redirect(postUrl, http.StatusSeeOther)
}

func ForumPostDelete(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	postData := FetchPostAndStuff(c.Context(), c.Conn, cd.ThreadID, cd.PostID)

	baseData := getBaseData(
		c,
		fmt.Sprintf("Deleting post in \"%s\" | %s", postData.Thread.Title, cd.SubforumTree[cd.SubforumID].Name),
		ForumThreadBreadcrumbs(cd.LineageBuilder, c.CurrentProject, &postData.Thread),
	)

	templatePost := templates.PostToTemplate(&postData.Post, postData.Author, c.Theme)
	templatePost.AddContentVersion(postData.CurrentVersion, postData.Editor)

	type forumPostDeleteData struct {
		templates.BaseData
		Post      templates.Post
		SubmitUrl string
	}

	var res ResponseData
	res.MustWriteTemplate("forum_post_delete.html", forumPostDeleteData{
		BaseData:  baseData,
		SubmitUrl: hmnurl.BuildForumPostDelete(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), cd.ThreadID, cd.PostID),
		Post:      templatePost,
	}, c.Perf)
	return res
}

func ForumPostDeleteSubmit(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	threadDeleted := DeletePost(c.Context(), tx, cd.ThreadID, cd.PostID)

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete post"))
	}

	if threadDeleted {
		forumUrl := hmnurl.BuildForum(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), 1)
		return c.Redirect(forumUrl, http.StatusSeeOther)
	} else {
		threadUrl := hmnurl.BuildForumThread(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), cd.ThreadID, "", 1) // TODO: Go to the last page of the thread? Or the post before the post we just deleted?
		return c.Redirect(threadUrl, http.StatusSeeOther)
	}
}

type commonForumData struct {
	c *RequestContext

	SubforumID int
	ThreadID   int
	PostID     int

	SubforumTree   models.SubforumTree
	LineageBuilder *models.SubforumLineageBuilder
}

/*
Gets data that is used on basically every forums-related route. Parses path params for subforum,
thread, and post ids and validates that all those resources do in fact exist.

Returns false if any data is invalid and you should return a 404.
*/
func getCommonForumData(c *RequestContext) (commonForumData, bool) {
	c.Perf.StartBlock("FORUMS", "Fetch common forum data")
	defer c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	res := commonForumData{
		c:              c,
		SubforumTree:   subforumTree,
		LineageBuilder: lineageBuilder,
	}

	if subforums, hasSubforums := c.PathParams["subforums"]; hasSubforums {
		sfId, valid := validateSubforums(lineageBuilder, c.CurrentProject, subforums)
		if !valid {
			return commonForumData{}, false
		}
		res.SubforumID = sfId

		// No need to validate that subforum exists here; it's handled by validateSubforums.
	}

	if threadIdStr, hasThreadId := c.PathParams["threadid"]; hasThreadId {
		threadId, err := strconv.Atoi(threadIdStr)
		if err != nil {
			return commonForumData{}, false
		}
		res.ThreadID = threadId

		c.Perf.StartBlock("SQL", "Verify that the thread exists")
		threadExists, err := db.QueryBool(c.Context(), c.Conn,
			`
			SELECT COUNT(*) > 0
			FROM handmade_thread
			WHERE
				id = $1
				AND subforum_id = $2
				AND NOT deleted
			`,
			res.ThreadID,
			res.SubforumID,
		)
		c.Perf.EndBlock()
		if err != nil {
			panic(err)
		}
		if !threadExists {
			return commonForumData{}, false
		}
	}

	if postIdStr, hasPostId := c.PathParams["postid"]; hasPostId {
		postId, err := strconv.Atoi(postIdStr)
		if err != nil {
			return commonForumData{}, false
		}
		res.PostID = postId

		c.Perf.StartBlock("SQL", "Verify that the post exists")
		postExists, err := db.QueryBool(c.Context(), c.Conn,
			`
			SELECT COUNT(*) > 0
			FROM handmade_post
			WHERE
				id = $1
				AND thread_id = $2
				AND NOT deleted
			`,
			res.PostID,
			res.ThreadID,
		)
		c.Perf.EndBlock()
		if err != nil {
			panic(err)
		}
		if !postExists {
			return commonForumData{}, false
		}
	}

	return res, true
}

func validateSubforums(lineageBuilder *models.SubforumLineageBuilder, project *models.Project, sfPath string) (int, bool) {
	if project.ForumID == nil {
		return -1, false
	}

	subforumId := *project.ForumID
	if len(sfPath) == 0 {
		return subforumId, true
	}

	sfPath = strings.ToLower(sfPath)
	valid := false
	sfSlugs := strings.Split(sfPath, "/")
	lastSlug := sfSlugs[len(sfSlugs)-1]
	if len(lastSlug) > 0 {
		lastSlugSfId := lineageBuilder.FindIdBySlug(project.ID, lastSlug)
		if lastSlugSfId != -1 {
			subforumSlugs := lineageBuilder.GetSubforumLineageSlugs(lastSlugSfId)
			allMatch := true
			for i, subforum := range subforumSlugs {
				if subforum != sfSlugs[i] {
					allMatch = false
					break
				}
			}
			valid = allMatch
		}
		if valid {
			subforumId = lastSlugSfId
		}
	}
	return subforumId, valid
}

func addForumUrlsToPost(p *templates.Post, projectSlug string, subforums []string, threadId int, postId int) {
	p.Url = hmnurl.BuildForumPost(projectSlug, subforums, threadId, postId)
	p.DeleteUrl = hmnurl.BuildForumPostDelete(projectSlug, subforums, threadId, postId)
	p.EditUrl = hmnurl.BuildForumPostEdit(projectSlug, subforums, threadId, postId)
	p.ReplyUrl = hmnurl.BuildForumPostReply(projectSlug, subforums, threadId, postId)
}

func addAuthorCountsToPost(ctx context.Context, conn db.ConnOrTx, p *templates.Post) {
	numPosts, err := db.QueryInt(ctx, conn,
		`
		SELECT COUNT(*)
		FROM
			handmade_post AS post
			JOIN handmade_project AS project ON post.project_id = project.id
		WHERE
			post.author_id = $1
			AND NOT post.deleted
			AND project.lifecycle = ANY ($2)
		`,
		p.Author.ID,
		models.VisibleProjectLifecycles,
	)
	if err != nil {
		logging.ExtractLogger(ctx).Warn().Err(err).Msg("failed to get count of user posts")
	} else {
		p.AuthorNumPosts = numPosts
	}

	numProjects, err := db.QueryInt(ctx, conn,
		`
		SELECT COUNT(*)
		FROM
			handmade_project AS project
			JOIN handmade_user_projects AS uproj ON uproj.project_id = project.id
		WHERE
			project.lifecycle = ANY ($1)
			AND uproj.user_id = $2
		`,
		models.VisibleProjectLifecycles,
		p.Author.ID,
	)
	if err != nil {
		logging.ExtractLogger(ctx).Warn().Err(err).Msg("failed to get count of user projects")
	} else {
		p.AuthorNumProjects = numProjects
	}
}

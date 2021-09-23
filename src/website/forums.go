package website

import (
	"context"
	"errors"
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

	MaxFileSize int
	UploadUrl   string
}

func getEditorDataForNew(currentUser *models.User, baseData templates.BaseData, replyPost *templates.Post) editorData {
	result := editorData{
		BaseData:       baseData,
		CanEditTitle:   replyPost == nil,
		PostReplyingTo: replyPost,
		MaxFileSize:    AssetMaxSize(currentUser),
		UploadUrl:      hmnurl.BuildAssetUpload(baseData.Project.Subdomain),
	}

	if replyPost != nil {
		result.Title = "Replying to post"
	}

	return result
}

func getEditorDataForEdit(currentUser *models.User, baseData templates.BaseData, p PostAndStuff) editorData {
	return editorData{
		BaseData:            baseData,
		Title:               p.Thread.Title,
		CanEditTitle:        p.Thread.FirstID == p.Post.ID,
		IsEditing:           true,
		EditInitialContents: p.CurrentVersion.TextRaw,
		MaxFileSize:         AssetMaxSize(currentUser),
		UploadUrl:           hmnurl.BuildAssetUpload(baseData.Project.Subdomain),
	}
}

func Forum(c *RequestContext) ResponseData {
	const threadsPerPage = 25

	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	currentSubforumSlugs := cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID)

	numThreads, err := CountThreads(c.Context(), c.Conn, c.CurrentUser, ThreadsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
		SubforumIDs: []int{cd.SubforumID},
	})
	if err != nil {
		panic(oops.New(err, "failed to get count of threads"))
	}

	numPages := utils.NumPages(numThreads, threadsPerPage)
	page, ok := ParsePageNumber(c, "page", numPages)
	if !ok {
		c.Redirect(hmnurl.BuildForum(c.CurrentProject.Slug, currentSubforumSlugs, page), http.StatusSeeOther)
	}
	howManyThreadsToSkip := (page - 1) * threadsPerPage

	mainThreads, err := FetchThreads(c.Context(), c.Conn, c.CurrentUser, ThreadsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
		SubforumIDs: []int{cd.SubforumID},
		Limit:       threadsPerPage,
		Offset:      howManyThreadsToSkip,
	})

	makeThreadListItem := func(row ThreadAndStuff) templates.ThreadListItem {
		return templates.ThreadListItem{
			Title:     row.Thread.Title,
			Url:       hmnurl.BuildForumThread(c.CurrentProject.Slug, cd.LineageBuilder.GetSubforumLineageSlugs(*row.Thread.SubforumID), row.Thread.ID, row.Thread.Title, 1),
			FirstUser: templates.UserToTemplate(row.FirstPostAuthor, c.Theme),
			FirstDate: row.FirstPost.PostDate,
			LastUser:  templates.UserToTemplate(row.LastPostAuthor, c.Theme),
			LastDate:  row.LastPost.PostDate,
			Unread:    row.Unread,
		}
	}

	var threads []templates.ThreadListItem
	for _, row := range mainThreads {
		threads = append(threads, makeThreadListItem(row))
	}

	// ---------------------
	// Subforum things
	// ---------------------

	var subforums []forumSubforumData
	if page == 1 {
		subforumNodes := cd.SubforumTree[cd.SubforumID].Children

		for _, sfNode := range subforumNodes {
			numThreads, err := CountThreads(c.Context(), c.Conn, c.CurrentUser, ThreadsQuery{
				ProjectIDs:  []int{c.CurrentProject.ID},
				ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
				SubforumIDs: []int{sfNode.ID},
			})
			if err != nil {
				panic(oops.New(err, "failed to get count of threads"))
			}

			subforumThreads, err := FetchThreads(c.Context(), c.Conn, c.CurrentUser, ThreadsQuery{
				ProjectIDs:  []int{c.CurrentProject.ID},
				ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
				SubforumIDs: []int{sfNode.ID},
				Limit:       3,
			})

			var threads []templates.ThreadListItem
			for _, row := range subforumThreads {
				threads = append(threads, makeThreadListItem(row))
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

// How many posts to display on a single page of a forum thread.
var threadPostsPerPage = 15

func ForumThread(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	currentSubforumSlugs := cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID)

	threads, err := FetchThreads(c.Context(), c.Conn, c.CurrentUser, ThreadsQuery{
		ProjectIDs: []int{c.CurrentProject.ID},
		ThreadIDs:  []int{cd.ThreadID},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get thread"))
	}
	if len(threads) == 0 {
		return FourOhFour(c)
	}
	threadResult := threads[0]
	thread := threadResult.Thread

	numPosts, err := CountPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
		ThreadIDs:   []int{cd.ThreadID},
	})
	if err != nil {
		panic(oops.New(err, "failed to get count of posts for thread"))
	}
	page, numPages, ok := getPageInfo(c.PathParams["page"], numPosts, threadPostsPerPage)
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

	postsAndStuff, err := FetchPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
		ProjectIDs: []int{c.CurrentProject.ID},
		ThreadIDs:  []int{thread.ID},
		Limit:      threadPostsPerPage,
		Offset:     (page - 1) * threadPostsPerPage,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch thread posts"))
	}

	var posts []templates.Post
	for _, p := range postsAndStuff {
		post := templates.PostToTemplate(&p.Post, p.Author, c.Theme)
		post.AddContentVersion(p.CurrentVersion, p.Editor)
		addForumUrlsToPost(&post, c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, post.ID)

		if p.ReplyPost != nil {
			reply := templates.PostToTemplate(p.ReplyPost, p.ReplyAuthor, c.Theme)
			addForumUrlsToPost(&reply, c.CurrentProject.Slug, currentSubforumSlugs, thread.ID, reply.ID)
			post.ReplyPost = &reply
		}

		addAuthorCountsToPost(c.Context(), c.Conn, &post)

		posts = append(posts, post)
	}

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
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    threadResult.FirstPost.Preview,
	})

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

	posts, err := FetchPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
		ThreadIDs:   []int{cd.ThreadID},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch posts for redirect"))
	}

	var post PostAndStuff
	postIdx := -1
	for i, p := range posts {
		if p.Post.ID == cd.PostID {
			post = p
			postIdx = i
			break
		}
	}
	if postIdx == -1 {
		return FourOhFour(c)
	}

	page := (postIdx / threadPostsPerPage) + 1

	return c.Redirect(hmnurl.BuildForumThreadWithPostHash(
		c.CurrentProject.Slug,
		cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID),
		cd.ThreadID,
		post.Thread.Title,
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
	editData := getEditorDataForNew(c.CurrentUser, baseData, nil)
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

	post, err := FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch post for reply"))
	}

	baseData := getBaseData(
		c,
		fmt.Sprintf("Replying to post | %s", cd.SubforumTree[cd.SubforumID].Name),
		ForumThreadBreadcrumbs(cd.LineageBuilder, c.CurrentProject, &post.Thread),
	)

	replyPost := templates.PostToTemplate(&post.Post, post.Author, c.Theme)
	replyPost.AddContentVersion(post.CurrentVersion, post.Editor)

	editData := getEditorDataForNew(c.CurrentUser, baseData, &replyPost)
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

	post, err := FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	}

	// Replies to the OP should not be considered replies
	var replyPostId *int
	if cd.PostID != post.Thread.FirstID {
		replyPostId = &cd.PostID
	}

	newPostId, _ := CreateNewPost(c.Context(), tx, c.CurrentProject.ID, cd.ThreadID, models.ThreadTypeForumPost, c.CurrentUser.ID, replyPostId, unparsed, c.Req.Host)

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

	post, err := FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch post for editing"))
	}

	title := ""
	if post.Thread.FirstID == post.Post.ID {
		title = fmt.Sprintf("Editing \"%s\" | %s", post.Thread.Title, cd.SubforumTree[cd.SubforumID].Name)
	} else {
		title = fmt.Sprintf("Editing Post | %s", cd.SubforumTree[cd.SubforumID].Name)
	}
	baseData := getBaseData(c, title, ForumThreadBreadcrumbs(cd.LineageBuilder, c.CurrentProject, &post.Thread))

	editData := getEditorDataForEdit(c.CurrentUser, baseData, post)
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

	post, err := FetchThreadPost(c.Context(), tx, c.CurrentUser, cd.ThreadID, cd.PostID, PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get forum post to submit edits"))
	}

	c.Req.ParseForm()
	title := c.Req.Form.Get("title")
	unparsed := c.Req.Form.Get("body")
	editReason := c.Req.Form.Get("editreason")
	if title != "" && post.Thread.FirstID != post.Post.ID {
		return RejectRequest(c, "You can only edit the title by editing the first post.")
	}
	if unparsed == "" {
		return RejectRequest(c, "You must provide a body for your post.")
	}

	CreatePostVersion(c.Context(), tx, cd.PostID, unparsed, c.Req.Host, editReason, &c.CurrentUser.ID)

	if title != "" {
		_, err := tx.Exec(c.Context(),
			`
			UPDATE handmade_thread SET title = $1 WHERE id = $2
			`,
			title,
			post.Thread.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update thread title"))
		}
	}

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

	post, err := FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch post for delete"))
	}

	baseData := getBaseData(
		c,
		fmt.Sprintf("Deleting post in \"%s\" | %s", post.Thread.Title, cd.SubforumTree[cd.SubforumID].Name),
		ForumThreadBreadcrumbs(cd.LineageBuilder, c.CurrentProject, &post.Thread),
	)

	templatePost := templates.PostToTemplate(&post.Post, post.Author, c.Theme)
	templatePost.AddContentVersion(post.CurrentVersion, post.Editor)

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

func WikiArticleRedirect(c *RequestContext) ResponseData {
	threadIdStr := c.PathParams["threadid"]
	threadId, err := strconv.Atoi(threadIdStr)
	if err != nil {
		panic(err)
	}

	thread, err := FetchThread(c.Context(), c.Conn, c.CurrentUser, threadId, ThreadsQuery{
		ProjectIDs: []int{c.CurrentProject.ID},
		// This is the rare query where we want all thread types!
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up wiki thread"))
	}

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	dest := UrlForGenericThread(&thread.Thread, lineageBuilder, c.CurrentProject.Slug)
	return c.Redirect(dest, http.StatusFound)
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
thread, and post ids.

Does NOT validate that the requested thread and post ID are valid.

If this returns false, then something was malformed and you should 404.
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
	}

	if threadIdStr, hasThreadId := c.PathParams["threadid"]; hasThreadId {
		threadId, err := strconv.Atoi(threadIdStr)
		if err != nil {
			return commonForumData{}, false
		}
		res.ThreadID = threadId
	}

	if postIdStr, hasPostId := c.PathParams["postid"]; hasPostId {
		postId, err := strconv.Atoi(postIdStr)
		if err != nil {
			return commonForumData{}, false
		}
		res.PostID = postId
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

// Takes a template post and adds information about how many posts the user has made
// on the site.
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

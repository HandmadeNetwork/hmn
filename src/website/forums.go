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
	"git.handmade.network/hmn/hmn/src/hmndata"
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

type editorData struct {
	templates.BaseData
	SubmitUrl   string
	SubmitLabel string

	// The following are filled out automatically by the
	// getEditorDataFor* functions.
	PostTitle           string
	CanEditPostTitle    bool
	IsEditing           bool
	EditInitialContents string
	PostReplyingTo      *templates.Post
	ShowEduOptions      bool
	PreviewClass        string

	TextEditor templates.TextEditor
}

func getEditorDataForNew(urlContext *hmnurl.UrlContext, currentUser *models.User, baseData templates.BaseData, replyPost *templates.Post) editorData {
	result := editorData{
		BaseData:         baseData,
		CanEditPostTitle: replyPost == nil,
		PostReplyingTo:   replyPost,
		TextEditor: templates.TextEditor{
			MaxFileSize: AssetMaxSize(currentUser),
			UploadUrl:   urlContext.BuildAssetUpload(),
		},
	}

	if replyPost != nil {
		result.PostTitle = "Replying to post"
	}

	return result
}

func getEditorDataForEdit(urlContext *hmnurl.UrlContext, currentUser *models.User, baseData templates.BaseData, p hmndata.PostAndStuff) editorData {
	return editorData{
		BaseData:            baseData,
		PostTitle:           p.Thread.Title,
		CanEditPostTitle:    p.Thread.FirstID == p.Post.ID,
		IsEditing:           true,
		EditInitialContents: p.CurrentVersion.TextRaw,
		TextEditor: templates.TextEditor{
			MaxFileSize: AssetMaxSize(currentUser),
			UploadUrl:   urlContext.BuildAssetUpload(),
		},
	}
}

func Forum(c *RequestContext) ResponseData {
	const threadsPerPage = 25

	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	currentSubforumSlugs := cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID)

	numThreads, err := hmndata.CountThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
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
		return c.Redirect(c.UrlContext.BuildForum(currentSubforumSlugs, page), http.StatusSeeOther)
	}
	howManyThreadsToSkip := (page - 1) * threadsPerPage

	mainThreads, err := hmndata.FetchThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
		SubforumIDs: []int{cd.SubforumID},
		Limit:       threadsPerPage,
		Offset:      howManyThreadsToSkip,
	})

	makeThreadListItem := func(row hmndata.ThreadAndStuff) templates.ThreadListItem {
		return templates.ThreadListItem{
			Title:     row.Thread.Title,
			Url:       c.UrlContext.BuildForumThread(cd.LineageBuilder.GetSubforumLineageSlugs(*row.Thread.SubforumID), row.Thread.ID, row.Thread.Title, 1),
			FirstUser: templates.UserToTemplate(row.FirstPostAuthor),
			FirstDate: row.FirstPost.PostDate,
			LastUser:  templates.UserToTemplate(row.LastPostAuthor),
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
			numThreads, err := hmndata.CountThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
				ProjectIDs:  []int{c.CurrentProject.ID},
				ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
				SubforumIDs: []int{sfNode.ID},
			})
			if err != nil {
				panic(oops.New(err, "failed to get count of threads"))
			}

			subforumThreads, err := hmndata.FetchThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
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
				Url:          c.UrlContext.BuildForum(cd.LineageBuilder.GetSubforumLineageSlugs(sfNode.ID), 1),
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
		SubforumBreadcrumbs(c.UrlContext, cd.LineageBuilder, cd.SubforumID),
	)

	var res ResponseData
	res.MustWriteTemplate("forum.html", forumData{
		BaseData:     baseData,
		NewThreadUrl: c.UrlContext.BuildForumNewThread(currentSubforumSlugs, false),
		MarkReadUrl:  c.UrlContext.BuildForumMarkRead(cd.SubforumID),
		Threads:      threads,
		Pagination: templates.Pagination{
			Current: page,
			Total:   numPages,

			FirstUrl:    c.UrlContext.BuildForum(currentSubforumSlugs, 1),
			LastUrl:     c.UrlContext.BuildForum(currentSubforumSlugs, numPages),
			NextUrl:     c.UrlContext.BuildForum(currentSubforumSlugs, utils.Clamp(1, page+1, numPages)),
			PreviousUrl: c.UrlContext.BuildForum(currentSubforumSlugs, utils.Clamp(1, page-1, numPages)),
		},
		Subforums: subforums,
	}, c.Perf)
	return res
}

func ForumMarkRead(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c, c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	sfId, err := strconv.Atoi(c.PathParams["sfid"])
	if err != nil {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c)

	sfIds := []int{sfId}
	if sfId == 0 {
		// Mark literally everything as read
		_, err := tx.Exec(c,
			`
			UPDATE hmn_user
			SET marked_all_read_at = NOW()
			WHERE id = $1
			`,
			c.CurrentUser.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to mark all posts as read"))
		}

		// Delete thread unread info
		_, err = tx.Exec(c,
			`
			DELETE FROM thread_last_read_info
			WHERE user_id = $1;
			`,
			c.CurrentUser.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete thread unread info"))
		}

		// Delete subforum unread info
		_, err = tx.Exec(c,
			`
			DELETE FROM subforum_last_read_info
			WHERE user_id = $1;
			`,
			c.CurrentUser.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete subforum unread info"))
		}
	} else {
		c.Perf.StartBlock("SQL", "Update SLRIs")
		_, err = tx.Exec(c,
			`
		INSERT INTO subforum_last_read_info (subforum_id, user_id, lastread)
			SELECT id, $2, $3
			FROM subforum
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
		_, err = tx.Exec(c,
			`
		DELETE FROM thread_last_read_info
		WHERE
			user_id = $2
			AND thread_id IN (
				SELECT id
				FROM thread
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

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit SLRI/TLRI updates"))
	}

	var redirUrl string
	if sfId == 0 {
		redirUrl = hmnurl.BuildFeed()
	} else {
		redirUrl = c.UrlContext.BuildForum(lineageBuilder.GetSubforumLineageSlugs(sfId), 1)
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

	threads, err := hmndata.FetchThreads(c, c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
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
	currentSubforumSlugs := cd.LineageBuilder.GetSubforumLineageSlugs(*thread.SubforumID)

	if *thread.SubforumID != cd.SubforumID {
		correctThreadUrl := c.UrlContext.BuildForumThread(currentSubforumSlugs, thread.ID, thread.Title, 1)
		return c.Redirect(correctThreadUrl, http.StatusSeeOther)
	}

	numPosts, err := hmndata.CountPosts(c, c.Conn, c.CurrentUser, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
		ThreadIDs:   []int{cd.ThreadID},
	})
	if err != nil {
		panic(oops.New(err, "failed to get count of posts for thread"))
	}
	page, numPages, ok := getPageInfo(c.PathParams["page"], numPosts, threadPostsPerPage)
	if !ok {
		urlNoPage := c.UrlContext.BuildForumThread(currentSubforumSlugs, thread.ID, thread.Title, 1)
		return c.Redirect(urlNoPage, http.StatusSeeOther)
	}
	pagination := templates.Pagination{
		Current: page,
		Total:   numPages,

		FirstUrl:    c.UrlContext.BuildForumThread(currentSubforumSlugs, thread.ID, thread.Title, 1),
		LastUrl:     c.UrlContext.BuildForumThread(currentSubforumSlugs, thread.ID, thread.Title, numPages),
		NextUrl:     c.UrlContext.BuildForumThread(currentSubforumSlugs, thread.ID, thread.Title, utils.Clamp(1, page+1, numPages)),
		PreviousUrl: c.UrlContext.BuildForumThread(currentSubforumSlugs, thread.ID, thread.Title, utils.Clamp(1, page-1, numPages)),
	}

	postsAndStuff, err := hmndata.FetchPosts(c, c.Conn, c.CurrentUser, hmndata.PostsQuery{
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
		post := templates.PostToTemplate(&p.Post, p.Author)
		post.AddContentVersion(p.CurrentVersion, p.Editor)
		post.ThreadLocked = thread.Locked
		addForumUrlsToPost(c.UrlContext, &post, c.CurrentUser, p.Author, currentSubforumSlugs, thread.ID, post.ID)

		if p.ReplyPost != nil {
			reply := templates.PostToTemplate(p.ReplyPost, p.ReplyAuthor)
			reply.ThreadLocked = thread.Locked
			addForumUrlsToPost(c.UrlContext, &reply, c.CurrentUser, p.Author, currentSubforumSlugs, thread.ID, reply.ID)
			post.ReplyPost = &reply
		}

		addAuthorCountsToPost(c, c.Conn, &post)

		posts = append(posts, post)
	}

	// Update thread last read info
	if c.CurrentUser != nil {
		c.Perf.StartBlock("SQL", "Update TLRI")
		_, err = c.Conn.Exec(c,
			`
			INSERT INTO thread_last_read_info (thread_id, user_id, lastread)
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

	baseData := getBaseData(c, thread.Title, SubforumBreadcrumbs(c.UrlContext, cd.LineageBuilder, cd.SubforumID))
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    threadResult.FirstPost.Preview,
	})

	var res ResponseData
	res.MustWriteTemplate("forum_thread.html", forumThreadData{
		BaseData:    baseData,
		Thread:      templates.ThreadToTemplate(&thread),
		Posts:       posts,
		SubforumUrl: c.UrlContext.BuildForum(currentSubforumSlugs, 1),
		ReplyUrl:    c.UrlContext.BuildForumPostReply(currentSubforumSlugs, thread.ID, thread.FirstID),
		Pagination:  pagination,
	}, c.Perf)
	return res
}

func ForumPostRedirect(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	posts, err := hmndata.FetchPosts(c, c.Conn, c.CurrentUser, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
		ThreadIDs:   []int{cd.ThreadID},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch posts for redirect"))
	}

	var post hmndata.PostAndStuff
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

	return c.Redirect(c.UrlContext.BuildForumThreadWithPostHash(
		cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID),
		post.Thread.ID,
		post.Thread.Title,
		page,
		post.Post.ID,
	), http.StatusSeeOther)
}

func ForumNewThread(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	baseData := getBaseData(c, "Create New Thread", SubforumBreadcrumbs(c.UrlContext, cd.LineageBuilder, cd.SubforumID))
	editData := getEditorDataForNew(c.UrlContext, c.CurrentUser, baseData, nil)
	editData.SubmitUrl = c.UrlContext.BuildForumNewThread(cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), true)
	editData.SubmitLabel = "Post New Thread"

	var res ResponseData
	res.MustWriteTemplate("editor.html", editData, c.Perf)
	return res
}

func ForumNewThreadSubmit(c *RequestContext) ResponseData {
	tx, err := c.Conn.Begin(c)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c)

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
		return c.RejectRequest("You must provide a title for your post.")
	}
	if unparsed == "" {
		return c.RejectRequest("You must provide a body for your post.")
	}

	// Create thread
	var threadId int
	err = tx.QueryRow(c,
		`
		INSERT INTO thread (title, sticky, type, project_id, subforum_id, first_id, last_id)
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
	hmndata.CreateNewPost(c, tx, c.CurrentProject.ID, threadId, models.ThreadTypeForumPost, c.CurrentUser.ID, nil, unparsed, c.Req.Host)

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create new forum thread"))
	}

	newThreadUrl := c.UrlContext.BuildForumThread(cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), threadId, title, 1)
	return c.Redirect(newThreadUrl, http.StatusSeeOther)
}

func ForumPostReply(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	post, err := hmndata.FetchThreadPost(c, c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch post for reply"))
	}

	if *post.Thread.SubforumID != cd.SubforumID {
		correctUrl := c.UrlContext.BuildForumPostReply(cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID), post.Thread.ID, post.Post.ID)
		return c.Redirect(correctUrl, http.StatusSeeOther)
	}

	baseData := getBaseData(
		c,
		fmt.Sprintf("Replying to post | %s", cd.SubforumTree[*post.Thread.SubforumID].Name),
		ForumThreadBreadcrumbs(c.UrlContext, cd.LineageBuilder, &post.Thread),
	)

	replyPost := templates.PostToTemplate(&post.Post, post.Author)
	replyPost.AddContentVersion(post.CurrentVersion, post.Editor)

	editData := getEditorDataForNew(c.UrlContext, c.CurrentUser, baseData, &replyPost)
	editData.SubmitUrl = c.UrlContext.BuildForumPostReply(cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID), post.Thread.ID, post.Post.ID)
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

	tx, err := c.Conn.Begin(c)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c)

	err = c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, oops.New(nil, "the form data was invalid"))
	}
	unparsed := c.Req.Form.Get("body")
	if unparsed == "" {
		return c.RejectRequest("Your reply cannot be empty.")
	}

	post, err := hmndata.FetchThreadPost(c, c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	}

	// Replies to the OP should not be considered replies
	var replyPostId *int
	if post.Post.ID != post.Thread.FirstID {
		replyPostId = &post.Post.ID
	}

	newPostId, _ := hmndata.CreateNewPost(c, tx, c.CurrentProject.ID, post.Thread.ID, models.ThreadTypeForumPost, c.CurrentUser.ID, replyPostId, unparsed, c.Req.Host)

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to reply to forum post"))
	}

	newPostUrl := c.UrlContext.BuildForumPost(cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID), post.Thread.ID, newPostId)
	return c.Redirect(newPostUrl, http.StatusSeeOther)
}

func ForumPostEdit(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !hmndata.UserCanEditPost(c, c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	post, err := hmndata.FetchThreadPost(c, c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch post for editing"))
	}

	if *post.Thread.SubforumID != cd.SubforumID {
		correctUrl := c.UrlContext.BuildForumPostEdit(cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID), post.Thread.ID, post.Post.ID)
		return c.Redirect(correctUrl, http.StatusSeeOther)
	}

	title := ""
	if post.Thread.FirstID == post.Post.ID {
		title = fmt.Sprintf("Editing \"%s\" | %s", post.Thread.Title, cd.SubforumTree[*post.Thread.SubforumID].Name)
	} else {
		title = fmt.Sprintf("Editing Post | %s", cd.SubforumTree[*post.Thread.SubforumID].Name)
	}
	baseData := getBaseData(c, title, ForumThreadBreadcrumbs(c.UrlContext, cd.LineageBuilder, &post.Thread))

	editData := getEditorDataForEdit(c.UrlContext, c.CurrentUser, baseData, post)
	editData.SubmitUrl = c.UrlContext.BuildForumPostEdit(cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID), post.Thread.ID, post.Post.ID)
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

	if !hmndata.UserCanEditPost(c, c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c)

	post, err := hmndata.FetchThreadPost(c, tx, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
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
		return c.RejectRequest("You can only edit the title by editing the first post.")
	}
	if unparsed == "" {
		return c.RejectRequest("You must provide a body for your post.")
	}

	hmndata.CreatePostVersion(c, tx, post.Post.ID, unparsed, c.Req.Host, editReason, &c.CurrentUser.ID)

	if title != "" {
		_, err := tx.Exec(c,
			`
			UPDATE thread SET title = $1 WHERE id = $2
			`,
			title,
			post.Thread.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update thread title"))
		}
	}

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to edit forum post"))
	}

	postUrl := c.UrlContext.BuildForumPost(cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID), post.Thread.ID, post.Post.ID)
	return c.Redirect(postUrl, http.StatusSeeOther)
}

func ForumPostDelete(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !hmndata.UserCanEditPost(c, c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	post, err := hmndata.FetchThreadPost(c, c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeForumPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch post for delete"))
	}

	if *post.Thread.SubforumID != cd.SubforumID {
		correctUrl := c.UrlContext.BuildForumPostDelete(cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID), post.Thread.ID, post.Post.ID)
		return c.Redirect(correctUrl, http.StatusSeeOther)
	}

	baseData := getBaseData(
		c,
		fmt.Sprintf("Deleting post in \"%s\" | %s", post.Thread.Title, cd.SubforumTree[*post.Thread.SubforumID].Name),
		ForumThreadBreadcrumbs(c.UrlContext, cd.LineageBuilder, &post.Thread),
	)

	templatePost := templates.PostToTemplate(&post.Post, post.Author)
	templatePost.AddContentVersion(post.CurrentVersion, post.Editor)

	type forumPostDeleteData struct {
		templates.BaseData
		Post      templates.Post
		SubmitUrl string
	}

	var res ResponseData
	res.MustWriteTemplate("forum_post_delete.html", forumPostDeleteData{
		BaseData:  baseData,
		SubmitUrl: c.UrlContext.BuildForumPostDelete(cd.LineageBuilder.GetSubforumLineageSlugs(*post.Thread.SubforumID), post.Thread.ID, post.Post.ID),
		Post:      templatePost,
	}, c.Perf)
	return res
}

func ForumPostDeleteSubmit(c *RequestContext) ResponseData {
	cd, ok := getCommonForumData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !hmndata.UserCanEditPost(c, c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c)

	threadDeleted := hmndata.DeletePost(c, tx, cd.ThreadID, cd.PostID)

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete post"))
	}

	if threadDeleted {
		forumUrl := c.UrlContext.BuildForum(cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), 1)
		return c.Redirect(forumUrl, http.StatusSeeOther)
	} else {
		threadUrl := c.UrlContext.BuildForumThread(cd.LineageBuilder.GetSubforumLineageSlugs(cd.SubforumID), cd.ThreadID, "", 1) // TODO: Go to the last page of the thread? Or the post before the post we just deleted?
		return c.Redirect(threadUrl, http.StatusSeeOther)
	}
}

func WikiArticleRedirect(c *RequestContext) ResponseData {
	threadIdStr := c.PathParams["threadid"]
	threadId, err := strconv.Atoi(threadIdStr)
	if err != nil {
		panic(err)
	}

	thread, err := hmndata.FetchThread(c, c.Conn, c.CurrentUser, threadId, hmndata.ThreadsQuery{
		ProjectIDs: []int{c.CurrentProject.ID},
		// This is the rare query where we want all thread types!
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up wiki thread"))
	}

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c, c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	dest := UrlForGenericThread(c.UrlContext, &thread.Thread, lineageBuilder)
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
	subforumTree := models.GetFullSubforumTree(c, c.Conn)
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

func addForumUrlsToPost(urlContext *hmnurl.UrlContext, p *templates.Post, currentUser *models.User, author *models.User, subforums []string, threadId int, postId int) {
	p.Url = urlContext.BuildForumPost(subforums, threadId, postId)
	if currentUser != nil && ((author != nil && currentUser.ID == author.ID && !p.ThreadLocked) || currentUser.IsStaff) {
		p.DeleteUrl = urlContext.BuildForumPostDelete(subforums, threadId, postId)
		p.EditUrl = urlContext.BuildForumPostEdit(subforums, threadId, postId)
		p.ReplyUrl = urlContext.BuildForumPostReply(subforums, threadId, postId)
	}
}

// Takes a template post and adds information about how many posts the user has made
// on the site.
func addAuthorCountsToPost(ctx context.Context, conn db.ConnOrTx, p *templates.Post) {
	numPosts, err := db.QueryOneScalar[int](ctx, conn,
		`
		SELECT COUNT(*)
		FROM
			post
			JOIN project ON post.project_id = project.id
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

	numProjects, err := db.QueryOneScalar[int](ctx, conn,
		`
		SELECT COUNT(*)
		FROM
			project
			JOIN user_project AS uproj ON uproj.project_id = project.id
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

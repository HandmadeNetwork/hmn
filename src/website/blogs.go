package website

import (
	"fmt"
	"net/http"
	"strconv"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func BlogThread(c *RequestContext) ResponseData {
	type blogPostData struct {
		templates.BaseData
		Thread    templates.Thread
		MainPost  templates.Post
		Comments  []templates.Post
		ReplyLink string
		LoginLink string
	}

	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	thread, posts := FetchThreadPostsAndStuff(
		c.Context(),
		c.Conn,
		cd.ThreadID,
		0, 0,
	)

	var templatePosts []templates.Post
	for _, p := range posts {
		post := templates.PostToTemplate(&p.Post, p.Author, c.Theme)
		post.AddContentVersion(p.CurrentVersion, p.Editor)
		addBlogUrlsToPost(&post, c.CurrentProject.Slug, &p.Thread, p.Post.ID)

		if p.ReplyPost != nil {
			reply := templates.PostToTemplate(p.ReplyPost, p.ReplyAuthor, c.Theme)
			addBlogUrlsToPost(&reply, c.CurrentProject.Slug, &p.Thread, p.Post.ID)
			post.ReplyPost = &reply
		}

		templatePosts = append(templatePosts, post)
	}

	baseData := getBaseData(c)
	baseData.Title = thread.Title

	var res ResponseData
	res.MustWriteTemplate("blog_post.html", blogPostData{
		BaseData:  baseData,
		Thread:    templates.ThreadToTemplate(&thread),
		MainPost:  templatePosts[0],
		Comments:  templatePosts[1:],
		ReplyLink: hmnurl.BuildBlogPostReply(c.CurrentProject.Slug, cd.ThreadID, posts[0].Post.ID),
		LoginLink: hmnurl.BuildLoginPage(c.FullUrl()),
	}, c.Perf)
	return res
}

func BlogPostRedirectToThread(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	thread := FetchThread(c.Context(), c.Conn, cd.ThreadID)

	threadUrl := hmnurl.BuildBlogThread(c.CurrentProject.Slug, cd.ThreadID, thread.Title)
	return c.Redirect(threadUrl, http.StatusFound)
}

func BlogNewThread(c *RequestContext) ResponseData {
	baseData := getBaseData(c)
	baseData.Title = fmt.Sprintf("Create New Post | %s", c.CurrentProject.Name)
	baseData.MathjaxEnabled = true
	// TODO(ben): Set breadcrumbs

	editData := getEditorDataForNew(baseData, nil)
	editData.SubmitUrl = hmnurl.BuildBlogNewThread(c.CurrentProject.Slug)
	editData.SubmitLabel = "Create Post"

	var res ResponseData
	res.MustWriteTemplate("editor.html", editData, c.Perf)
	return res
}

func BlogNewThreadSubmit(c *RequestContext) ResponseData {
	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	err = c.Req.ParseForm()
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, oops.New(err, "the form data was invalid"))
	}
	title := c.Req.Form.Get("title")
	unparsed := c.Req.Form.Get("body")
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
		INSERT INTO handmade_thread (title, type, project_id, first_id, last_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
		`,
		title,
		models.ThreadTypeProjectBlogPost,
		c.CurrentProject.ID,
		-1,
		-1,
	).Scan(&threadId)
	if err != nil {
		panic(oops.New(err, "failed to create thread"))
	}

	// Create everything else
	CreateNewPost(c.Context(), tx, c.CurrentProject.ID, threadId, models.ThreadTypeProjectBlogPost, c.CurrentUser.ID, nil, unparsed, c.Req.Host)

	err = tx.Commit(c.Context())
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create new blog post"))
	}

	newThreadUrl := hmnurl.BuildBlogThread(c.CurrentProject.Slug, threadId, title)
	return c.Redirect(newThreadUrl, http.StatusSeeOther)
}

func BlogPostEdit(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	postData := FetchPostAndStuff(c.Context(), c.Conn, cd.ThreadID, cd.PostID)

	baseData := getBaseData(c)
	if postData.Thread.FirstID == postData.Post.ID {
		baseData.Title = fmt.Sprintf("Editing \"%s\" | %s", postData.Thread.Title, c.CurrentProject.Name)
	} else {
		baseData.Title = fmt.Sprintf("Editing Post | %s", c.CurrentProject.Name)
	}
	baseData.MathjaxEnabled = true
	// TODO(ben): Set breadcrumbs

	editData := getEditorDataForEdit(baseData, postData)
	editData.SubmitUrl = hmnurl.BuildBlogPostEdit(c.CurrentProject.Slug, cd.ThreadID, cd.PostID)
	editData.SubmitLabel = "Submit Edited Post"
	if postData.Thread.FirstID != postData.Post.ID {
		editData.SubmitLabel = "Submit Edited Comment"
	}

	var res ResponseData
	res.MustWriteTemplate("editor.html", editData, c.Perf)
	return res
}

func BlogPostEditSubmit(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
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

	postData := FetchPostAndStuff(c.Context(), tx, cd.ThreadID, cd.PostID)

	c.Req.ParseForm()
	title := c.Req.Form.Get("title")
	unparsed := c.Req.Form.Get("body")
	editReason := c.Req.Form.Get("editreason")
	if title != "" && postData.Thread.FirstID != postData.Post.ID {
		return RejectRequest(c, "You can only edit the title by editing the first post.")
	}
	if unparsed == "" {
		return RejectRequest(c, "You must provide a post body.")
	}

	CreatePostVersion(c.Context(), tx, postData.Post.ID, unparsed, c.Req.Host, editReason, &c.CurrentUser.ID)

	err = tx.Commit(c.Context())
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to edit blog post"))
	}

	postUrl := hmnurl.BuildBlogThreadWithPostHash(c.CurrentProject.Slug, cd.ThreadID, postData.Thread.Title, cd.PostID)
	return c.Redirect(postUrl, http.StatusSeeOther)
}

func BlogPostReply(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	postData := FetchPostAndStuff(c.Context(), c.Conn, cd.ThreadID, cd.PostID)

	baseData := getBaseData(c)
	baseData.Title = fmt.Sprintf("Replying to comment in \"%s\" | %s", postData.Thread.Title, c.CurrentProject.Name)
	baseData.MathjaxEnabled = true
	// TODO(ben): Set breadcrumbs

	replyPost := templates.PostToTemplate(&postData.Post, postData.Author, c.Theme)
	replyPost.AddContentVersion(postData.CurrentVersion, postData.Editor)

	editData := getEditorDataForNew(baseData, &replyPost)
	editData.SubmitUrl = hmnurl.BuildBlogPostReply(c.CurrentProject.Slug, cd.ThreadID, cd.PostID)
	editData.SubmitLabel = "Submit Reply"

	var res ResponseData
	res.MustWriteTemplate("editor.html", editData, c.Perf)
	return res
}

func BlogPostReplySubmit(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
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
		return ErrorResponse(http.StatusBadRequest, oops.New(nil, "the form data was invalid"))
	}
	unparsed := c.Req.Form.Get("body")
	if unparsed == "" {
		return RejectRequest(c, "Your reply cannot be empty.")
	}

	newPostId, _ := CreateNewPost(c.Context(), tx, c.CurrentProject.ID, cd.ThreadID, models.ThreadTypeProjectBlogPost, c.CurrentUser.ID, &cd.PostID, unparsed, c.Req.Host)

	err = tx.Commit(c.Context())
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to reply to blog post"))
	}

	newPostUrl := hmnurl.BuildBlogPost(c.CurrentProject.Slug, cd.ThreadID, newPostId)
	return c.Redirect(newPostUrl, http.StatusSeeOther)
}

func BlogPostDelete(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	postData := FetchPostAndStuff(c.Context(), c.Conn, cd.ThreadID, cd.PostID)

	baseData := getBaseData(c)
	if postData.Thread.FirstID == postData.Post.ID {
		baseData.Title = fmt.Sprintf("Deleting \"%s\" | %s", postData.Thread.Title, c.CurrentProject.Name)
	} else {
		baseData.Title = fmt.Sprintf("Deleting comment in \"%s\" | %s", postData.Thread.Title, c.CurrentProject.Name)
	}
	baseData.MathjaxEnabled = true
	// TODO(ben): Set breadcrumbs

	templatePost := templates.PostToTemplate(&postData.Post, postData.Author, c.Theme)
	templatePost.AddContentVersion(postData.CurrentVersion, postData.Editor)

	type blogPostDeleteData struct {
		templates.BaseData
		Post      templates.Post
		SubmitUrl string
	}

	var res ResponseData
	res.MustWriteTemplate("blog_post_delete.html", blogPostDeleteData{
		BaseData:  baseData,
		SubmitUrl: hmnurl.BuildBlogPostDelete(c.CurrentProject.Slug, cd.ThreadID, cd.PostID),
		Post:      templatePost,
	}, c.Perf)
	return res
}

func BlogPostDeleteSubmit(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
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
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete post"))
	}

	if threadDeleted {
		projectUrl := hmnurl.BuildProjectHomepage(c.CurrentProject.Slug)
		return c.Redirect(projectUrl, http.StatusSeeOther)
	} else {
		thread := FetchThread(c.Context(), c.Conn, cd.ThreadID)
		threadUrl := hmnurl.BuildBlogThread(c.CurrentProject.Slug, thread.ID, thread.Title)
		return c.Redirect(threadUrl, http.StatusSeeOther)
	}
}

type commonBlogData struct {
	c *RequestContext

	ThreadID int
	PostID   int
}

func getCommonBlogData(c *RequestContext) (commonBlogData, bool) {
	c.Perf.StartBlock("BLOGS", "Fetch common blog data")
	defer c.Perf.EndBlock()

	res := commonBlogData{
		c: c,
	}

	if threadIdStr, hasThreadId := c.PathParams["threadid"]; hasThreadId {
		threadId, err := strconv.Atoi(threadIdStr)
		if err != nil {
			return commonBlogData{}, false
		}
		res.ThreadID = threadId

		c.Perf.StartBlock("SQL", "Verify that the thread exists")
		threadExists, err := db.QueryBool(c.Context(), c.Conn,
			`
			SELECT COUNT(*) > 0
			FROM handmade_thread
			WHERE
				id = $1
				AND project_id = $2
			`,
			res.ThreadID,
			c.CurrentProject.ID,
		)
		c.Perf.EndBlock()
		if err != nil {
			panic(err)
		}
		if !threadExists {
			return commonBlogData{}, false
		}
	}

	if postIdStr, hasPostId := c.PathParams["postid"]; hasPostId {
		postId, err := strconv.Atoi(postIdStr)
		if err != nil {
			return commonBlogData{}, false
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
			`,
			res.PostID,
			res.ThreadID,
		)
		c.Perf.EndBlock()
		if err != nil {
			panic(err)
		}
		if !postExists {
			return commonBlogData{}, false
		}
	}

	return res, true
}

func addBlogUrlsToPost(p *templates.Post, projectSlug string, thread *models.Thread, postId int) {
	p.Url = hmnurl.BuildBlogThreadWithPostHash(projectSlug, thread.ID, thread.Title, postId)
	p.DeleteUrl = hmnurl.BuildBlogPostDelete(projectSlug, thread.ID, postId)
	p.EditUrl = hmnurl.BuildBlogPostEdit(projectSlug, thread.ID, postId)
	p.ReplyUrl = hmnurl.BuildBlogPostReply(projectSlug, thread.ID, postId)
}

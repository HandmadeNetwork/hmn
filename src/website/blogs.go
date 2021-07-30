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
		BaseData: baseData,
		Thread:   templates.ThreadToTemplate(&thread),
		MainPost: templatePosts[0],
		Comments: templatePosts[1:],
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

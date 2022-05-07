package website

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

func BlogIndex(c *RequestContext) ResponseData {
	type blogIndexEntry struct {
		Title   string
		Url     string
		Author  templates.User
		Date    time.Time
		Content template.HTML
	}
	type blogIndexData struct {
		templates.BaseData
		Posts      []blogIndexEntry
		Pagination templates.Pagination

		CanCreatePost bool
		NewPostUrl    string
	}

	const postsPerPage = 5

	numThreads, err := hmndata.CountThreads(c.Context(), c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch total number of blog posts"))
	}

	numPages := utils.NumPages(numThreads, postsPerPage)
	page, ok := ParsePageNumber(c, "page", numPages)
	if !ok {
		c.Redirect(c.UrlContext.BuildBlog(page), http.StatusSeeOther)
	}

	threads, err := hmndata.FetchThreads(c.Context(), c.Conn, c.CurrentUser, hmndata.ThreadsQuery{
		ProjectIDs:     []int{c.CurrentProject.ID},
		ThreadTypes:    []models.ThreadType{models.ThreadTypeProjectBlogPost},
		Limit:          postsPerPage,
		Offset:         (page - 1) * postsPerPage,
		OrderByCreated: true,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch blog posts for index"))
	}

	var entries []blogIndexEntry
	for _, thread := range threads {
		entries = append(entries, blogIndexEntry{
			Title:   thread.Thread.Title,
			Url:     c.UrlContext.BuildBlogThread(thread.Thread.ID, thread.Thread.Title),
			Author:  templates.UserToTemplate(thread.FirstPostAuthor, c.Theme),
			Date:    thread.FirstPost.PostDate,
			Content: template.HTML(thread.FirstPostCurrentVersion.TextParsed),
		})
	}

	baseData := getBaseData(c, fmt.Sprintf("%s Blog", c.CurrentProject.Name), []templates.Breadcrumb{BlogBreadcrumb(c.UrlContext)})

	canCreate := false
	if c.CurrentProject.HasBlog() && c.CurrentUser != nil {
		isProjectOwner := false
		owners, err := hmndata.FetchProjectOwners(c.Context(), c.Conn, c.CurrentUser, c.CurrentProject.ID)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch project owners"))
		}
		for _, owner := range owners {
			if owner.ID == c.CurrentUser.ID {
				isProjectOwner = true
				break
			}
		}

		canCreate = c.CurrentUser.IsStaff || isProjectOwner
	}

	var res ResponseData
	res.MustWriteTemplate("blog_index.html", blogIndexData{
		BaseData: baseData,
		Posts:    entries,
		Pagination: templates.Pagination{
			Current: page,
			Total:   numPages,

			FirstUrl:    c.UrlContext.BuildBlog(1),
			LastUrl:     c.UrlContext.BuildBlog(numPages),
			PreviousUrl: c.UrlContext.BuildBlog(utils.IntClamp(1, page-1, numPages)),
			NextUrl:     c.UrlContext.BuildBlog(utils.IntClamp(1, page+1, numPages)),
		},

		CanCreatePost: canCreate,
		NewPostUrl:    c.UrlContext.BuildBlogNewThread(),
	}, c.Perf)
	return res
}

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

	thread, posts, err := hmndata.FetchThreadPosts(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch posts for blog thread"))
	}

	var templatePosts []templates.Post
	for _, p := range posts {
		post := templates.PostToTemplate(&p.Post, p.Author, c.Theme)
		post.AddContentVersion(p.CurrentVersion, p.Editor)
		addBlogUrlsToPost(c.UrlContext, &post, &p.Thread, p.Post.ID)

		if p.ReplyPost != nil {
			reply := templates.PostToTemplate(p.ReplyPost, p.ReplyAuthor, c.Theme)
			addBlogUrlsToPost(c.UrlContext, &reply, &p.Thread, p.Post.ID)
			post.ReplyPost = &reply
		}

		templatePosts = append(templatePosts, post)
	}
	// Update thread last read info
	if c.CurrentUser != nil {
		c.Perf.StartBlock("SQL", "Update TLRI")
		_, err := c.Conn.Exec(c.Context(),
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
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update blog tlri"))
		}
	}

	baseData := getBaseData(c, thread.Title, []templates.Breadcrumb{BlogBreadcrumb(c.UrlContext)})
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    posts[0].Post.Preview,
	})

	var res ResponseData
	res.MustWriteTemplate("blog_post.html", blogPostData{
		BaseData:  baseData,
		Thread:    templates.ThreadToTemplate(&thread),
		MainPost:  templatePosts[0],
		Comments:  templatePosts[1:],
		ReplyLink: c.UrlContext.BuildBlogPostReply(cd.ThreadID, posts[0].Post.ID),
		LoginLink: hmnurl.BuildLoginPage(c.FullUrl()),
	}, c.Perf)
	return res
}

func BlogPostRedirectToThread(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	thread, err := hmndata.FetchThread(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, hmndata.ThreadsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch thread for blog redirect"))
	}

	threadUrl := c.UrlContext.BuildBlogThreadWithPostHash(cd.ThreadID, thread.Thread.Title, cd.PostID)
	return c.Redirect(threadUrl, http.StatusFound)
}

func BlogNewThread(c *RequestContext) ResponseData {
	baseData := getBaseData(
		c,
		fmt.Sprintf("Create New Post | %s", c.CurrentProject.Name),
		[]templates.Breadcrumb{BlogBreadcrumb(c.UrlContext)},
	)

	editData := getEditorDataForNew(c.UrlContext, c.CurrentUser, baseData, nil)
	editData.SubmitUrl = c.UrlContext.BuildBlogNewThread()
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
		return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "the form data was invalid"))
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
		INSERT INTO thread (title, type, project_id, first_id, last_id)
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
	hmndata.CreateNewPost(c.Context(), tx, c.CurrentProject.ID, threadId, models.ThreadTypeProjectBlogPost, c.CurrentUser.ID, nil, unparsed, c.Req.Host)

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create new blog post"))
	}

	newThreadUrl := c.UrlContext.BuildBlogThread(threadId, title)
	return c.Redirect(newThreadUrl, http.StatusSeeOther)
}

func BlogPostEdit(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !hmndata.UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	post, err := hmndata.FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get blog post to edit"))
	}

	title := ""
	if post.Thread.FirstID == post.Post.ID {
		title = fmt.Sprintf("Editing \"%s\" | %s", post.Thread.Title, c.CurrentProject.Name)
	} else {
		title = fmt.Sprintf("Editing Post | %s", c.CurrentProject.Name)
	}
	baseData := getBaseData(
		c,
		title,
		BlogThreadBreadcrumbs(c.UrlContext, &post.Thread),
	)

	editData := getEditorDataForEdit(c.UrlContext, c.CurrentUser, baseData, post)
	editData.SubmitUrl = c.UrlContext.BuildBlogPostEdit(cd.ThreadID, cd.PostID)
	editData.SubmitLabel = "Submit Edited Post"
	if post.Thread.FirstID != post.Post.ID {
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

	if !hmndata.UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	post, err := hmndata.FetchThreadPost(c.Context(), tx, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get blog post to submit edits"))
	}

	c.Req.ParseForm()
	title := c.Req.Form.Get("title")
	unparsed := c.Req.Form.Get("body")
	editReason := c.Req.Form.Get("editreason")
	if title != "" && post.Thread.FirstID != post.Post.ID {
		return RejectRequest(c, "You can only edit the title by editing the first post.")
	}
	if unparsed == "" {
		return RejectRequest(c, "You must provide a post body.")
	}

	hmndata.CreatePostVersion(c.Context(), tx, post.Post.ID, unparsed, c.Req.Host, editReason, &c.CurrentUser.ID)

	if title != "" {
		_, err := tx.Exec(c.Context(),
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

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to edit blog post"))
	}

	postUrl := c.UrlContext.BuildBlogThreadWithPostHash(cd.ThreadID, post.Thread.Title, cd.PostID)
	return c.Redirect(postUrl, http.StatusSeeOther)
}

func BlogPostReply(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	post, err := hmndata.FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get blog post for reply"))
	}

	baseData := getBaseData(
		c,
		fmt.Sprintf("Replying to comment in \"%s\" | %s", post.Thread.Title, c.CurrentProject.Name),
		BlogThreadBreadcrumbs(c.UrlContext, &post.Thread),
	)

	replyPost := templates.PostToTemplate(&post.Post, post.Author, c.Theme)
	replyPost.AddContentVersion(post.CurrentVersion, post.Editor)

	editData := getEditorDataForNew(c.UrlContext, c.CurrentUser, baseData, &replyPost)
	editData.SubmitUrl = c.UrlContext.BuildBlogPostReply(cd.ThreadID, cd.PostID)
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
		return c.ErrorResponse(http.StatusBadRequest, oops.New(nil, "the form data was invalid"))
	}
	unparsed := c.Req.Form.Get("body")
	if unparsed == "" {
		return RejectRequest(c, "Your reply cannot be empty.")
	}

	newPostId, _ := hmndata.CreateNewPost(c.Context(), tx, c.CurrentProject.ID, cd.ThreadID, models.ThreadTypeProjectBlogPost, c.CurrentUser.ID, &cd.PostID, unparsed, c.Req.Host)

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to reply to blog post"))
	}

	newPostUrl := c.UrlContext.BuildBlogPost(cd.ThreadID, newPostId)
	return c.Redirect(newPostUrl, http.StatusSeeOther)
}

func BlogPostDelete(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !hmndata.UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	post, err := hmndata.FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, hmndata.PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get blog post to delete"))
	}

	title := ""
	if post.Thread.FirstID == post.Post.ID {
		title = fmt.Sprintf("Deleting \"%s\" | %s", post.Thread.Title, c.CurrentProject.Name)
	} else {
		title = fmt.Sprintf("Deleting comment in \"%s\" | %s", post.Thread.Title, c.CurrentProject.Name)
	}
	baseData := getBaseData(
		c,
		title,
		BlogThreadBreadcrumbs(c.UrlContext, &post.Thread),
	)

	templatePost := templates.PostToTemplate(&post.Post, post.Author, c.Theme)
	templatePost.AddContentVersion(post.CurrentVersion, post.Editor)

	type blogPostDeleteData struct {
		templates.BaseData
		Post      templates.Post
		SubmitUrl string
	}

	var res ResponseData
	res.MustWriteTemplate("blog_post_delete.html", blogPostDeleteData{
		BaseData:  baseData,
		SubmitUrl: c.UrlContext.BuildBlogPostDelete(cd.ThreadID, cd.PostID),
		Post:      templatePost,
	}, c.Perf)
	return res
}

func BlogPostDeleteSubmit(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	if !hmndata.UserCanEditPost(c.Context(), c.Conn, *c.CurrentUser, cd.PostID) {
		return FourOhFour(c)
	}

	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	threadDeleted := hmndata.DeletePost(c.Context(), tx, cd.ThreadID, cd.PostID)

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete post"))
	}

	if threadDeleted {
		return c.Redirect(c.UrlContext.BuildHomepage(), http.StatusSeeOther)
	} else {
		thread, err := hmndata.FetchThread(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, hmndata.ThreadsQuery{
			ProjectIDs:  []int{c.CurrentProject.ID},
			ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
		})
		if errors.Is(err, db.NotFound) {
			panic(oops.New(err, "the thread was supposedly not deleted after deleting a post in a blog, but the thread was not found afterwards"))
		} else if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch thread after blog post delete"))
		}
		threadUrl := c.UrlContext.BuildBlogThread(thread.Thread.ID, thread.Thread.Title)
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
		threadExists, err := db.QueryOneScalar[bool](c.Context(), c.Conn,
			`
			SELECT COUNT(*) > 0
			FROM thread
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
		postExists, err := db.QueryOneScalar[bool](c.Context(), c.Conn,
			`
			SELECT COUNT(*) > 0
			FROM post
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

func addBlogUrlsToPost(urlContext *hmnurl.UrlContext, p *templates.Post, thread *models.Thread, postId int) {
	p.Url = urlContext.BuildBlogThreadWithPostHash(thread.ID, thread.Title, postId)
	p.DeleteUrl = urlContext.BuildBlogPostDelete(thread.ID, postId)
	p.EditUrl = urlContext.BuildBlogPostEdit(thread.ID, postId)
	p.ReplyUrl = urlContext.BuildBlogPostReply(thread.ID, postId)
}

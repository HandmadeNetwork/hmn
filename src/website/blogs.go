package website

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
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

	numPosts, err := CountPosts(c.Context(), c.Conn, c.CurrentUser, PostsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch total number of blog posts"))
	}

	numPages := utils.NumPages(numPosts, postsPerPage)
	page, ok := ParsePageNumber(c, "page", numPages)
	if !ok {
		c.Redirect(hmnurl.BuildBlog(c.CurrentProject.Slug, page), http.StatusSeeOther)
	}

	threads, err := FetchThreads(c.Context(), c.Conn, c.CurrentUser, ThreadsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
		Limit:       postsPerPage,
		Offset:      (page - 1) * postsPerPage,
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch blog posts for index"))
	}

	var entries []blogIndexEntry
	for _, thread := range threads {
		entries = append(entries, blogIndexEntry{
			Title:   thread.Thread.Title,
			Url:     hmnurl.BuildBlogThread(c.CurrentProject.Slug, thread.Thread.ID, thread.Thread.Title),
			Author:  templates.UserToTemplate(thread.FirstPostAuthor, c.Theme),
			Date:    thread.FirstPost.PostDate,
			Content: template.HTML(thread.FirstPostCurrentVersion.TextParsed),
		})
	}

	baseData := getBaseData(c, fmt.Sprintf("%s Blog", c.CurrentProject.Name), []templates.Breadcrumb{BlogBreadcrumb(c.CurrentProject.Slug)})

	canCreate := false
	if c.CurrentUser != nil {
		isProjectOwner := false
		owners, err := FetchProjectOwners(c.Context(), c.Conn, c.CurrentProject.ID)
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

			FirstUrl:    hmnurl.BuildBlog(c.CurrentProject.Slug, 1),
			LastUrl:     hmnurl.BuildBlog(c.CurrentProject.Slug, numPages),
			PreviousUrl: hmnurl.BuildBlog(c.CurrentProject.Slug, utils.IntClamp(1, page-1, numPages)),
			NextUrl:     hmnurl.BuildBlog(c.CurrentProject.Slug, utils.IntClamp(1, page+1, numPages)),
		},

		CanCreatePost: canCreate,
		NewPostUrl:    hmnurl.BuildBlogNewThread(c.CurrentProject.Slug),
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

	thread, posts, err := FetchThreadPosts(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, PostsQuery{
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
		addBlogUrlsToPost(&post, c.CurrentProject.Slug, &p.Thread, p.Post.ID)

		if p.ReplyPost != nil {
			reply := templates.PostToTemplate(p.ReplyPost, p.ReplyAuthor, c.Theme)
			addBlogUrlsToPost(&reply, c.CurrentProject.Slug, &p.Thread, p.Post.ID)
			post.ReplyPost = &reply
		}

		templatePosts = append(templatePosts, post)
	}
	// Update thread last read info
	if c.CurrentUser != nil {
		c.Perf.StartBlock("SQL", "Update TLRI")
		_, err := c.Conn.Exec(c.Context(),
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
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update blog tlri"))
		}
	}

	baseData := getBaseData(c, thread.Title, []templates.Breadcrumb{BlogBreadcrumb(c.CurrentProject.Slug)})
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

	thread, err := FetchThread(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, ThreadsQuery{
		ProjectIDs:  []int{c.CurrentProject.ID},
		ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
	})
	if errors.Is(err, db.NotFound) {
		return FourOhFour(c)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch thread for blog redirect"))
	}

	threadUrl := hmnurl.BuildBlogThreadWithPostHash(c.CurrentProject.Slug, cd.ThreadID, thread.Thread.Title, cd.PostID)
	return c.Redirect(threadUrl, http.StatusFound)
}

func BlogNewThread(c *RequestContext) ResponseData {
	baseData := getBaseData(
		c,
		fmt.Sprintf("Create New Post | %s", c.CurrentProject.Name),
		[]templates.Breadcrumb{BlogBreadcrumb(c.CurrentProject.Slug)},
	)

	editData := getEditorDataForNew(c.CurrentUser, baseData, nil)
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
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create new blog post"))
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

	post, err := FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, PostsQuery{
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
		BlogThreadBreadcrumbs(c.CurrentProject.Slug, &post.Thread),
	)

	editData := getEditorDataForEdit(c.CurrentUser, baseData, post)
	editData.SubmitUrl = hmnurl.BuildBlogPostEdit(c.CurrentProject.Slug, cd.ThreadID, cd.PostID)
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

	CreatePostVersion(c.Context(), tx, post.Post.ID, unparsed, c.Req.Host, editReason, &c.CurrentUser.ID)

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
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to edit blog post"))
	}

	postUrl := hmnurl.BuildBlogThreadWithPostHash(c.CurrentProject.Slug, cd.ThreadID, post.Thread.Title, cd.PostID)
	return c.Redirect(postUrl, http.StatusSeeOther)
}

func BlogPostReply(c *RequestContext) ResponseData {
	cd, ok := getCommonBlogData(c)
	if !ok {
		return FourOhFour(c)
	}

	post, err := FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, PostsQuery{
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
		BlogThreadBreadcrumbs(c.CurrentProject.Slug, &post.Thread),
	)

	replyPost := templates.PostToTemplate(&post.Post, post.Author, c.Theme)
	replyPost.AddContentVersion(post.CurrentVersion, post.Editor)

	editData := getEditorDataForNew(c.CurrentUser, baseData, &replyPost)
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
		return c.ErrorResponse(http.StatusBadRequest, oops.New(nil, "the form data was invalid"))
	}
	unparsed := c.Req.Form.Get("body")
	if unparsed == "" {
		return RejectRequest(c, "Your reply cannot be empty.")
	}

	newPostId, _ := CreateNewPost(c.Context(), tx, c.CurrentProject.ID, cd.ThreadID, models.ThreadTypeProjectBlogPost, c.CurrentUser.ID, &cd.PostID, unparsed, c.Req.Host)

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to reply to blog post"))
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

	post, err := FetchThreadPost(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, cd.PostID, PostsQuery{
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
		BlogThreadBreadcrumbs(c.CurrentProject.Slug, &post.Thread),
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
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete post"))
	}

	if threadDeleted {
		projectUrl := UrlForProject(c.CurrentProject)
		return c.Redirect(projectUrl, http.StatusSeeOther)
	} else {
		thread, err := FetchThread(c.Context(), c.Conn, c.CurrentUser, cd.ThreadID, ThreadsQuery{
			ProjectIDs:  []int{c.CurrentProject.ID},
			ThreadTypes: []models.ThreadType{models.ThreadTypeProjectBlogPost},
		})
		if errors.Is(err, db.NotFound) {
			panic(oops.New(err, "the thread was supposedly not deleted after deleting a post in a blog, but the thread was not found afterwards"))
		} else if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch thread after blog post delete"))
		}
		threadUrl := hmnurl.BuildBlogThread(c.CurrentProject.Slug, thread.Thread.ID, thread.Thread.Title)
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

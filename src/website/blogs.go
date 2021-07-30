package website

import (
	"net/http"
	"strconv"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
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
		addBlogUrlsToPost(&post, c.CurrentProject.Slug, p.Thread.ID, p.Post.ID)

		if p.ReplyPost != nil {
			reply := templates.PostToTemplate(p.ReplyPost, p.ReplyAuthor, c.Theme)
			addBlogUrlsToPost(&reply, c.CurrentProject.Slug, p.Thread.ID, p.Post.ID)
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

	threadUrl := hmnurl.BuildBlogThread(c.CurrentProject.Slug, cd.ThreadID, thread.Title, 1)
	return c.Redirect(threadUrl, http.StatusFound)
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

func addBlogUrlsToPost(p *templates.Post, projectSlug string, threadId int, postId int) {
	p.Url = hmnurl.BuildBlogPost(projectSlug, threadId, postId)
	p.DeleteUrl = hmnurl.BuildBlogPostDelete(projectSlug, threadId, postId)
	p.EditUrl = hmnurl.BuildBlogPostEdit(projectSlug, threadId, postId)
	p.ReplyUrl = hmnurl.BuildBlogPostReply(projectSlug, threadId, postId)
}

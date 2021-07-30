package website

import (
	"context"
	"errors"
	"math"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

type postAndRelatedModels struct {
	Thread         models.Thread
	Post           models.Post
	CurrentVersion models.PostVersion

	Author *models.User
	Editor *models.User

	ReplyPost   *models.Post
	ReplyAuthor *models.User
}

/*
Fetches the thread defined by your (already parsed) path params.

YOU MUST VERIFY THAT THE THREAD ID IS VALID BEFORE CALLING THIS FUNCTION. It will
not check, for example, that the thread belongs to the correct subforum.
*/
func FetchThread(ctx context.Context, connOrTx db.ConnOrTx, threadId int) models.Thread {
	type threadQueryResult struct {
		Thread models.Thread `db:"thread"`
	}
	irow, err := db.QueryOne(ctx, connOrTx, threadQueryResult{},
		`
		SELECT $columns
		FROM
			handmade_thread AS thread
		WHERE
			id = $1
			AND NOT deleted
		`,
		threadId,
	)
	if err != nil {
		// We shouldn't encounter db.ErrNoMatchingRows, because validation should have verified that everything exists.
		panic(oops.New(err, "failed to fetch thread"))
	}

	thread := irow.(*threadQueryResult).Thread
	return thread
}

/*
Fetches the post, the thread, and author / editor information for the post defined in
your path params.

YOU MUST VERIFY THAT THE THREAD ID AND POST ID ARE VALID BEFORE CALLING THIS FUNCTION.
It will not check that the post belongs to the correct subforum, for example, or the
correct project blog. This logic varies per route and per use of threads, so it doesn't
happen here.
*/
func FetchPostAndStuff(
	ctx context.Context,
	connOrTx db.ConnOrTx,
	threadId, postId int,
) postAndRelatedModels {
	type resultRow struct {
		Thread         models.Thread      `db:"thread"`
		Post           models.Post        `db:"post"`
		CurrentVersion models.PostVersion `db:"ver"`
		Author         *models.User       `db:"author"`
		Editor         *models.User       `db:"editor"`
		ReplyPost      *models.Post       `db:"reply"`
		ReplyAuthor    *models.User       `db:"reply_author"`
	}
	postQueryResult, err := db.QueryOne(ctx, connOrTx, resultRow{},
		`
		SELECT $columns
		FROM
			handmade_thread AS thread
			JOIN handmade_post AS post ON post.thread_id = thread.id
			JOIN handmade_postversion AS ver ON post.current_id = ver.id
			LEFT JOIN auth_user AS author ON post.author_id = author.id
			LEFT JOIN auth_user AS editor ON ver.editor_id = editor.id
			LEFT JOIN handmade_post AS reply ON post.reply_id = reply.id
			LEFT JOIN auth_user AS reply_author ON reply.author_id = reply_author.id
		WHERE
			post.thread_id = $1
			AND post.id = $2
			AND NOT post.deleted
		`,
		threadId,
		postId,
	)
	if err != nil {
		// We shouldn't encounter db.ErrNoMatchingRows, because validation should have verified that everything exists.
		panic(oops.New(err, "failed to fetch post and related data"))
	}

	result := postQueryResult.(*resultRow)
	return postAndRelatedModels{
		Thread:         result.Thread,
		Post:           result.Post,
		CurrentVersion: result.CurrentVersion,
		Author:         result.Author,
		Editor:         result.Editor,
		ReplyPost:      result.ReplyPost,
		ReplyAuthor:    result.ReplyAuthor,
	}
}

/*
Fetches all the posts (and related models) for a given thread.

YOU MUST VERIFY THAT THE THREAD ID IS VALID BEFORE CALLING THIS FUNCTION. It will
not check, for example, that the thread belongs to the correct subforum.
*/
func FetchThreadPostsAndStuff(
	ctx context.Context,
	connOrTx db.ConnOrTx,
	threadId int,
	page, postsPerPage int,
) (models.Thread, []postAndRelatedModels) {
	limit := postsPerPage
	offset := (page - 1) * postsPerPage
	if postsPerPage == 0 {
		limit = math.MaxInt32
		offset = 0
	}

	thread := FetchThread(ctx, connOrTx, threadId)

	type postResult struct {
		Post           models.Post        `db:"post"`
		CurrentVersion models.PostVersion `db:"ver"`
		Author         *models.User       `db:"author"`
		Editor         *models.User       `db:"editor"`
		ReplyPost      *models.Post       `db:"reply"`
		ReplyAuthor    *models.User       `db:"reply_author"`
	}
	itPosts, err := db.Query(ctx, connOrTx, postResult{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			JOIN handmade_postversion AS ver ON post.current_id = ver.id
			LEFT JOIN auth_user AS author ON post.author_id = author.id
			LEFT JOIN auth_user AS editor ON ver.editor_id = editor.id
			LEFT JOIN handmade_post AS reply ON post.reply_id = reply.id
			LEFT JOIN auth_user AS reply_author ON reply.author_id = reply_author.id
		WHERE
			post.thread_id = $1
			AND NOT post.deleted
		ORDER BY post.postdate
		LIMIT $2 OFFSET $3
		`,
		thread.ID,
		limit,
		offset,
	)
	if err != nil {
		panic(oops.New(err, "failed to fetch posts for thread"))
	}
	defer itPosts.Close()

	var posts []postAndRelatedModels
	for {
		irow, hasNext := itPosts.Next()
		if !hasNext {
			break
		}

		row := irow.(*postResult)
		posts = append(posts, postAndRelatedModels{
			Thread:         thread,
			Post:           row.Post,
			CurrentVersion: row.CurrentVersion,
			Author:         row.Author,
			Editor:         row.Editor,
			ReplyPost:      row.ReplyPost,
			ReplyAuthor:    row.ReplyAuthor,
		})
	}

	return thread, posts
}

func (cd *commonForumData) UserCanEditPost(ctx context.Context, connOrTx db.ConnOrTx, user models.User) bool {
	if user.IsStaff {
		return true
	}

	type postResult struct {
		AuthorID *int `db:"post.author_id"`
	}
	iresult, err := db.QueryOne(ctx, connOrTx, postResult{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
		WHERE
			post.id = $1
			AND NOT post.deleted
		`,
		cd.PostID,
	)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return false
		} else {
			panic(oops.New(err, "failed to get author of post when checking permissions"))
		}
	}
	result := iresult.(*postResult)

	return result.AuthorID != nil && *result.AuthorID == user.ID
}

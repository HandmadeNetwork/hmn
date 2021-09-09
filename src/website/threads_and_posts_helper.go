package website

import (
	"context"
	"errors"
	"math"
	"net"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
	"github.com/jackc/pgx/v4"
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
) (models.Thread, []postAndRelatedModels, string) {
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

	preview, err := db.QueryString(ctx, connOrTx,
		`
		SELECT post.preview
		FROM
			handmade_post AS post
			JOIN handmade_thread AS thread ON post.thread_id = thread.id
			JOIN handmade_postversion AS ver ON post.current_id = ver.id
		WHERE
			post.thread_id = $1
			AND thread.first_id = post.id
		`,
		thread.ID,
	)
	if err != nil && !errors.Is(err, db.ErrNoMatchingRows) {
		panic(oops.New(err, "failed to fetch posts for thread"))
	}

	return thread, posts, preview
}

func UserCanEditPost(ctx context.Context, connOrTx db.ConnOrTx, user models.User, postId int) bool {
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
		postId,
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

func CreateNewPost(
	ctx context.Context,
	tx pgx.Tx,
	projectId int,
	threadId int, threadType models.ThreadType,
	userId int,
	replyId *int,
	unparsedContent string,
	ipString string,
) (postId, versionId int) {
	// Create post
	err := tx.QueryRow(ctx,
		`
		INSERT INTO handmade_post (postdate, thread_id, thread_type, current_id, author_id, project_id, reply_id, preview)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id
		`,
		time.Now(),
		threadId,
		threadType,
		-1,
		userId,
		projectId,
		replyId,
		"", // empty preview, will be updated later
	).Scan(&postId)
	if err != nil {
		panic(oops.New(err, "failed to create post"))
	}

	// Create and associate version
	versionId = CreatePostVersion(ctx, tx, postId, unparsedContent, ipString, "", nil)

	// Fix up thread
	err = FixThreadPostIds(ctx, tx, threadId)
	if err != nil {
		panic(oops.New(err, "failed to fix up thread post IDs"))
	}

	return
}

func DeletePost(
	ctx context.Context,
	tx pgx.Tx,
	threadId, postId int,
) (threadDeleted bool) {
	isFirstPost, err := db.QueryBool(ctx, tx,
		`
		SELECT thread.first_id = $1
		FROM
			handmade_thread AS thread
		WHERE
			thread.id = $2
		`,
		postId,
		threadId,
	)
	if err != nil {
		panic(oops.New(err, "failed to check if post was the first post in the thread"))
	}

	if isFirstPost {
		// Just delete the whole thread and all its posts.
		_, err = tx.Exec(ctx,
			`
			UPDATE handmade_thread
			SET deleted = TRUE
			WHERE id = $1
			`,
			threadId,
		)
		_, err = tx.Exec(ctx,
			`
			UPDATE handmade_post
			SET deleted = TRUE
			WHERE thread_id = $1
			`,
			threadId,
		)

		return true
	}

	_, err = tx.Exec(ctx,
		`
		UPDATE handmade_post
		SET deleted = TRUE
		WHERE
			id = $1
		`,
		postId,
	)
	if err != nil {
		panic(oops.New(err, "failed to mark forum post as deleted"))
	}

	err = FixThreadPostIds(ctx, tx, threadId)
	if err != nil {
		if errors.Is(err, errThreadEmpty) {
			panic("it shouldn't be possible to delete the last remaining post in a thread, without it also being the first post in the thread and thus resulting in the whole thread getting deleted earlier")
		} else {
			panic(oops.New(err, "failed to fix up thread post ids"))
		}
	}

	return false
}

func CreatePostVersion(ctx context.Context, tx pgx.Tx, postId int, unparsedContent string, ipString string, editReason string, editorId *int) (versionId int) {
	parsed := parsing.ParseMarkdown(unparsedContent, parsing.ForumRealMarkdown)
	ip := net.ParseIP(ipString)

	const previewMaxLength = 100
	parsedPlaintext := parsing.ParseMarkdown(unparsedContent, parsing.PlaintextMarkdown)
	preview := parsedPlaintext
	if len(preview) > previewMaxLength-1 {
		preview = preview[:previewMaxLength-1] + "…"
	}

	// Create post version
	err := tx.QueryRow(ctx,
		`
		INSERT INTO handmade_postversion (post_id, text_raw, text_parsed, ip, date, edit_reason, editor_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id
		`,
		postId,
		unparsedContent,
		parsed,
		ip,
		time.Now(),
		editReason,
		editorId,
	).Scan(&versionId)
	if err != nil {
		panic(oops.New(err, "failed to create post version"))
	}

	// Update post with version id and preview
	_, err = tx.Exec(ctx,
		`
		UPDATE handmade_post
		SET current_id = $1, preview = $2
		WHERE id = $3
		`,
		versionId,
		preview,
		postId,
	)
	if err != nil {
		panic(oops.New(err, "failed to set current post version and preview"))
	}

	return
}

var errThreadEmpty = errors.New("thread contained no non-deleted posts")

/*
Ensures that the first_id and last_id on the thread are still good.

Returns errThreadEmpty if the thread contains no visible posts any more.
You should probably mark the thread as deleted in this case.
*/
func FixThreadPostIds(ctx context.Context, tx pgx.Tx, threadId int) error {
	postsIter, err := db.Query(ctx, tx, models.Post{},
		`
		SELECT $columns
		FROM handmade_post
		WHERE
			thread_id = $1
			AND NOT deleted
		`,
		threadId,
	)
	if err != nil {
		return oops.New(err, "failed to fetch posts when fixing up thread")
	}

	var firstPost, lastPost *models.Post
	for _, ipost := range postsIter.ToSlice() {
		post := ipost.(*models.Post)

		if firstPost == nil || post.PostDate.Before(firstPost.PostDate) {
			firstPost = post
		}
		if lastPost == nil || post.PostDate.After(lastPost.PostDate) {
			lastPost = post
		}
	}

	if firstPost == nil || lastPost == nil {
		return errThreadEmpty
	}

	_, err = tx.Exec(ctx,
		`
		UPDATE handmade_thread
		SET first_id = $1, last_id = $2
		WHERE id = $3
		`,
		firstPost.ID,
		lastPost.ID,
		threadId,
	)
	if err != nil {
		return oops.New(err, "failed to update thread first/last ids")
	}

	return nil
}

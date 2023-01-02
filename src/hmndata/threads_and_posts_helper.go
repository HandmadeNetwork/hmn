package hmndata

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
	"git.handmade.network/hmn/hmn/src/perf"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type ThreadsQuery struct {
	// Available on all thread queries.
	ProjectIDs  []int               // if empty, all projects
	ThreadTypes []models.ThreadType // if empty, all types (you do not want to do this)
	SubforumIDs []int               // if empty, all subforums

	// Ignored when using FetchThread.
	ThreadIDs []int

	// Ignored when using FetchThread or CountThreads.
	Limit, Offset  int  // if empty, no pagination
	OrderByCreated bool // defaults to order by last updated
}

type ThreadAndStuff struct {
	Project                 models.Project     `db:"project"`
	Thread                  models.Thread      `db:"thread"`
	FirstPost               models.Post        `db:"first_post"`
	LastPost                models.Post        `db:"last_post"`
	FirstPostCurrentVersion models.PostVersion `db:"first_version"`
	LastPostCurrentVersion  models.PostVersion `db:"last_version"`
	FirstPostAuthor         *models.User       `db:"first_author"` // Can be nil in case of a deleted user
	LastPostAuthor          *models.User       `db:"last_author"`  // Can be nil in case of a deleted user
	Unread                  bool
}

/*
Fetches threads and related models from the database according to all the given
query params. For the most correct results, provide as much information as you have
on hand.
*/
func FetchThreads(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q ThreadsQuery,
) ([]ThreadAndStuff, error) {
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch threads")
	defer perf.EndBlock()

	var qb db.QueryBuilder

	var currentUserID *int
	if currentUser != nil {
		currentUserID = &currentUser.ID
	}

	qb.Add(
		`
		SELECT $columns
		FROM
			thread
			JOIN project ON thread.project_id = project.id
			JOIN post AS first_post ON first_post.id = thread.first_id
			JOIN post AS last_post ON last_post.id = thread.last_id
			JOIN post_version AS first_version ON first_version.id = first_post.current_id
			JOIN post_version AS last_version ON last_version.id = last_post.current_id
			LEFT JOIN hmn_user AS first_author ON first_author.id = first_post.author_id
			LEFT JOIN asset AS first_author_avatar ON first_author_avatar.id = first_author.avatar_asset_id
			LEFT JOIN hmn_user AS last_author ON last_author.id = last_post.author_id
			LEFT JOIN asset AS last_author_avatar ON last_author_avatar.id = last_author.avatar_asset_id
			LEFT JOIN thread_last_read_info AS tlri ON (
				tlri.thread_id = thread.id
				AND tlri.user_id = $?
			)
			LEFT JOIN subforum_last_read_info AS slri ON (
				slri.subforum_id = thread.subforum_id
				AND slri.user_id = $?
			)
		WHERE
			NOT thread.deleted
			AND ( -- project has valid lifecycle
				NOT project.hidden AND project.lifecycle = ANY($?)
				OR project.id = $?
			)
		`,
		currentUserID,
		currentUserID,
		models.VisibleProjectLifecycles,
		models.HMNProjectID,
	)
	if len(q.ProjectIDs) > 0 {
		qb.Add(`AND project.id = ANY ($?)`, q.ProjectIDs)
	}
	if len(q.ThreadTypes) > 0 {
		qb.Add(`AND thread.type = ANY ($?)`, q.ThreadTypes)
	}
	if len(q.SubforumIDs) > 0 {
		qb.Add(`AND thread.subforum_id = ANY ($?)`, q.SubforumIDs)
	}
	if len(q.ThreadIDs) > 0 {
		qb.Add(`AND thread.id = ANY ($?)`, q.ThreadIDs)
	}
	if currentUser == nil {
		qb.Add(
			`AND first_author.status = $? -- thread author is Approved`,
			models.UserStatusApproved,
		)
	} else if !currentUser.IsStaff {
		qb.Add(
			`
			AND (
				first_author.status = $? -- thread author is Approved
				OR first_author.id = $? -- current user is the thread author
			)
			`,
			models.UserStatusApproved,
			currentUserID,
		)
	}
	if q.OrderByCreated {
		qb.Add(`ORDER BY first_post.postdate DESC`)
	} else {
		qb.Add(`ORDER BY last_post.postdate DESC`)
	}
	if q.Limit > 0 {
		qb.Add(`LIMIT $? OFFSET $?`, q.Limit, q.Offset)
	}

	type resultRow struct {
		ThreadAndStuff
		FirstPostAuthorAvatar *models.Asset `db:"first_author_avatar"`
		LastPostAuthorAvatar  *models.Asset `db:"last_author_avatar"`
		ThreadLastReadTime    *time.Time    `db:"tlri.lastread"`
		ForumLastReadTime     *time.Time    `db:"slri.lastread"`
	}

	rows, err := db.Query[resultRow](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch threads")
	}

	result := make([]ThreadAndStuff, len(rows))
	for i, row := range rows {
		if row.FirstPostAuthor != nil {
			row.FirstPostAuthor.AvatarAsset = row.FirstPostAuthorAvatar
		}
		if row.LastPostAuthor != nil {
			row.LastPostAuthor.AvatarAsset = row.LastPostAuthorAvatar
		}

		hasRead := false
		if currentUser != nil && currentUser.MarkedAllReadAt.After(row.LastPost.PostDate) {
			hasRead = true
		} else if row.ThreadLastReadTime != nil && row.ThreadLastReadTime.After(row.LastPost.PostDate) {
			hasRead = true
		} else if row.ForumLastReadTime != nil && row.ForumLastReadTime.After(row.LastPost.PostDate) {
			hasRead = true
		}
		row.Unread = !hasRead

		result[i] = row.ThreadAndStuff
	}

	return result, nil
}

/*
Fetches a single thread and related data. A wrapper around FetchThreads.
As with FetchThreads, provide as much information as you know to get the
most correct results.

Returns db.NotFound if no result is found.
*/
func FetchThread(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	threadID int,
	q ThreadsQuery,
) (ThreadAndStuff, error) {
	q.ThreadIDs = []int{threadID}
	q.Limit = 1
	q.Offset = 0

	res, err := FetchThreads(ctx, dbConn, currentUser, q)
	if err != nil {
		return ThreadAndStuff{}, oops.New(err, "failed to fetch thread")
	}

	if len(res) == 0 {
		return ThreadAndStuff{}, db.NotFound
	}

	return res[0], nil
}

func CountThreads(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q ThreadsQuery,
) (int, error) {
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Count threads")
	defer perf.EndBlock()

	var qb db.QueryBuilder

	var currentUserID *int
	if currentUser != nil {
		currentUserID = &currentUser.ID
	}

	qb.Add(
		`
		SELECT COUNT(*)
		FROM
			thread
			JOIN project ON thread.project_id = project.id
			JOIN post AS first_post ON first_post.id = thread.first_id
			LEFT JOIN hmn_user AS first_author ON first_author.id = first_post.author_id
			LEFT JOIN asset AS first_author_avatar ON first_author_avatar.id = first_author.avatar_asset_id
		WHERE
			NOT thread.deleted
			AND ( -- project has valid lifecycle
				NOT project.hidden AND project.lifecycle = ANY($?)
				OR project.id = $?
			)
		`,
		models.VisibleProjectLifecycles,
		models.HMNProjectID,
	)
	if len(q.ProjectIDs) > 0 {
		qb.Add(`AND project.id = ANY ($?)`, q.ProjectIDs)
	}
	if len(q.ThreadTypes) > 0 {
		qb.Add(`AND thread.type = ANY ($?)`, q.ThreadTypes)
	}
	if len(q.SubforumIDs) > 0 {
		qb.Add(`AND thread.subforum_id = ANY ($?)`, q.SubforumIDs)
	}
	if currentUser == nil {
		qb.Add(
			`AND first_author.status = $? -- thread author is Approved`,
			models.UserStatusApproved,
		)
	} else if !currentUser.IsStaff {
		qb.Add(
			`
			AND (
				first_author.status = $? -- thread author is Approved
				OR first_author.id = $? -- current user is the thread author
			)
			`,
			models.UserStatusApproved,
			currentUserID,
		)
	}

	count, err := db.QueryOneScalar[int](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return 0, oops.New(err, "failed to fetch count of threads")
	}

	return count, nil
}

type PostsQuery struct {
	// Available on all post queries.
	ProjectIDs  []int
	UserIDs     []int
	ThreadTypes []models.ThreadType

	// Ignored when using FetchPost.
	ThreadIDs []int
	PostIDs   []int

	// Ignored when using FetchPost or CountPosts.
	Limit, Offset  int
	SortDescending bool
}

type PostAndStuff struct {
	Project        models.Project `db:"project"`
	Thread         models.Thread  `db:"thread"`
	Unread         bool
	Post           models.Post        `db:"post"`
	CurrentVersion models.PostVersion `db:"ver"`
	Author         *models.User       `db:"author"` // Can be nil in case of a deleted user
	Editor         *models.User       `db:"editor"`
	ReplyPost      *models.Post       `db:"reply_post"`
	ReplyAuthor    *models.User       `db:"reply_author"`
}

/*
Fetches posts and related models from the database according to all the given
query params. For the most correct results, provide as much information as you have
on hand.
*/
func FetchPosts(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q PostsQuery,
) ([]PostAndStuff, error) {
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch posts")
	defer perf.EndBlock()

	var qb db.QueryBuilder

	var currentUserID *int
	if currentUser != nil {
		currentUserID = &currentUser.ID
	}

	type resultRow struct {
		PostAndStuff
		AuthorAvatar       *models.Asset `db:"author_avatar"`
		EditorAvatar       *models.Asset `db:"editor_avatar"`
		ReplyAuthorAvatar  *models.Asset `db:"reply_author_avatar"`
		ThreadLastReadTime *time.Time    `db:"tlri.lastread"`
		ForumLastReadTime  *time.Time    `db:"slri.lastread"`
	}

	qb.Add(
		`
		SELECT $columns
		FROM
			post
			JOIN thread ON post.thread_id = thread.id
			JOIN project ON post.project_id = project.id
			JOIN post_version AS ver ON ver.id = post.current_id
			LEFT JOIN hmn_user AS author ON author.id = post.author_id
			LEFT JOIN asset AS author_avatar ON author_avatar.id = author.avatar_asset_id
			LEFT JOIN hmn_user AS editor ON ver.editor_id = editor.id
			LEFT JOIN asset AS editor_avatar ON editor_avatar.id = editor.avatar_asset_id
			LEFT JOIN thread_last_read_info AS tlri ON (
				tlri.thread_id = thread.id
				AND tlri.user_id = $?
			)
			LEFT JOIN subforum_last_read_info AS slri ON (
				slri.subforum_id = thread.subforum_id
				AND slri.user_id = $?
			)
			-- Unconditionally fetch reply info, but make sure to check it
			-- later and possibly remove these fields if the permission
			-- check fails.
			LEFT JOIN post AS reply_post ON reply_post.id = post.reply_id
			LEFT JOIN hmn_user AS reply_author ON reply_post.author_id = reply_author.id
			LEFT JOIN asset AS reply_author_avatar ON reply_author_avatar.id = reply_author.avatar_asset_id
		WHERE
			NOT thread.deleted
			AND NOT post.deleted
			AND ( -- project has valid lifecycle
				NOT project.hidden AND project.lifecycle = ANY($?)
				OR project.id = $?
			)
		`,
		currentUserID,
		currentUserID,
		models.VisibleProjectLifecycles,
		models.HMNProjectID,
	)
	if len(q.ProjectIDs) > 0 {
		qb.Add(`AND project.id = ANY ($?)`, q.ProjectIDs)
	}
	if len(q.UserIDs) > 0 {
		qb.Add(`AND post.author_id = ANY ($?)`, q.UserIDs)
	}
	if len(q.ThreadIDs) > 0 {
		qb.Add(`AND post.thread_id = ANY ($?)`, q.ThreadIDs)
	}
	if len(q.ThreadTypes) > 0 {
		qb.Add(`AND thread.type = ANY ($?)`, q.ThreadTypes)
	}
	if len(q.PostIDs) > 0 {
		qb.Add(`AND post.id = ANY ($?)`, q.PostIDs)
	}
	if currentUser == nil {
		qb.Add(
			`AND author.status = $? -- post author is Approved`,
			models.UserStatusApproved,
		)
	} else if !currentUser.IsStaff {
		qb.Add(
			`
			AND (
				author.status = $? -- post author is Approved
				OR author.id = $? -- current user is the post author
			)
			`,
			models.UserStatusApproved,
			currentUserID,
		)
	}
	qb.Add(`ORDER BY post.postdate`)
	if q.SortDescending {
		qb.Add(`DESC`)
	}
	if q.Limit > 0 {
		qb.Add(`LIMIT $? OFFSET $?`, q.Limit, q.Offset)
	}

	rows, err := db.Query[resultRow](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch posts")
	}

	result := make([]PostAndStuff, len(rows))
	for i, row := range rows {
		if row.Author != nil {
			row.Author.AvatarAsset = row.AuthorAvatar
		}
		if row.Editor != nil {
			row.Editor.AvatarAsset = row.EditorAvatar
		}
		if row.ReplyAuthor != nil {
			row.ReplyAuthor.AvatarAsset = row.ReplyAuthorAvatar
		}

		hasRead := false
		if currentUser != nil && currentUser.MarkedAllReadAt.After(row.Post.PostDate) {
			hasRead = true
		} else if row.ThreadLastReadTime != nil && row.ThreadLastReadTime.After(row.Post.PostDate) {
			hasRead = true
		} else if row.ForumLastReadTime != nil && row.ForumLastReadTime.After(row.Post.PostDate) {
			hasRead = true
		}
		row.Unread = !hasRead

		if row.ReplyPost != nil && row.ReplyAuthor != nil {
			replyAuthorIsNotApproved := row.ReplyAuthor.Status != models.UserStatusApproved
			canSeeUnapprovedReply := currentUser != nil && (row.ReplyAuthor.ID == currentUser.ID || currentUser.IsStaff)
			if replyAuthorIsNotApproved && !canSeeUnapprovedReply {
				row.ReplyPost = nil
				row.ReplyAuthor = nil
			}
		}

		result[i] = row.PostAndStuff
	}

	return result, nil
}

/*
Fetches posts for a given thread. A convenient wrapper around FetchPosts that returns
the posts and the actual thread model.

Return db.NotFound if nothing is found (no thread or no posts).
*/
func FetchThreadPosts(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	threadID int,
	q PostsQuery,
) (models.Thread, []PostAndStuff, error) {
	q.ThreadIDs = []int{threadID}

	res, err := FetchPosts(ctx, dbConn, currentUser, q)
	if err != nil {
		return models.Thread{}, nil, oops.New(err, "failed to fetch posts for thread")
	}

	if len(res) == 0 {
		// We shouldn't have threads without posts anyway.
		return models.Thread{}, nil, db.NotFound
	}

	return res[0].Thread, res, nil
}

/*
Fetches a single post for a thread and its related data. A wrapper
around FetchPosts. As with FetchPosts, provide as much information
as you know to get the most correct results.

Returns db.NotFound if no result is found.
*/
func FetchThreadPost(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	threadID, postID int,
	q PostsQuery,
) (PostAndStuff, error) {
	q.ThreadIDs = []int{threadID}
	q.PostIDs = []int{postID}
	q.Limit = 1
	q.Offset = 0

	res, err := FetchPosts(ctx, dbConn, currentUser, q)
	if err != nil {
		return PostAndStuff{}, oops.New(err, "failed to fetch post")
	}

	if len(res) == 0 {
		return PostAndStuff{}, db.NotFound
	}

	return res[0], nil
}

/*
Fetches a single post and its related data. A wrapper
around FetchPosts. As with FetchPosts, provide as much information
as you know to get the most correct results.

Returns db.NotFound if no result is found.
*/
func FetchPost(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	postID int,
	q PostsQuery,
) (PostAndStuff, error) {
	q.PostIDs = []int{postID}
	q.Limit = 1
	q.Offset = 0

	res, err := FetchPosts(ctx, dbConn, currentUser, q)
	if err != nil {
		return PostAndStuff{}, oops.New(err, "failed to fetch post")
	}

	if len(res) == 0 {
		return PostAndStuff{}, db.NotFound
	}

	return res[0], nil
}

func CountPosts(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q PostsQuery,
) (int, error) {
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Count posts")
	defer perf.EndBlock()

	var qb db.QueryBuilder

	var currentUserID *int
	if currentUser != nil {
		currentUserID = &currentUser.ID
	}

	qb.Add(
		`
		SELECT COUNT(*)
		FROM
			post
			JOIN thread ON post.thread_id = thread.id
			JOIN project ON post.project_id = project.id
			LEFT JOIN hmn_user AS author ON author.id = post.author_id
			LEFT JOIN asset AS author_avatar ON author_avatar.id = author.avatar_asset_id
		WHERE
			NOT thread.deleted
			AND NOT post.deleted
			AND ( -- project has valid lifecycle
				NOT project.hidden AND project.lifecycle = ANY($?)
				OR project.id = $?
			)
		`,
		models.VisibleProjectLifecycles,
		models.HMNProjectID,
	)
	if len(q.ProjectIDs) > 0 {
		qb.Add(`AND project.id = ANY ($?)`, q.ProjectIDs)
	}
	if len(q.UserIDs) > 0 {
		qb.Add(`AND post.author_id = ANY ($?)`, q.UserIDs)
	}
	if len(q.ThreadIDs) > 0 {
		qb.Add(`AND post.thread_id = ANY ($?)`, q.ThreadIDs)
	}
	if len(q.ThreadTypes) > 0 {
		qb.Add(`AND thread.type = ANY ($?)`, q.ThreadTypes)
	}
	if currentUser == nil {
		qb.Add(
			`AND author.status = $? -- post author is Approved`,
			models.UserStatusApproved,
		)
	} else if !currentUser.IsStaff {
		qb.Add(
			`
			AND (
				author.status = $? -- post author is Approved
				OR author.id = $? -- current user is the post author
			)
			`,
			models.UserStatusApproved,
			currentUserID,
		)
	}

	count, err := db.QueryOneScalar[int](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return 0, oops.New(err, "failed to count posts")
	}

	return count, nil
}

func UserCanEditPost(ctx context.Context, connOrTx db.ConnOrTx, user models.User, postId int) bool {
	if user.IsStaff {
		return true
	}

	authorID, err := db.QueryOneScalar[*int](ctx, connOrTx,
		`
		SELECT post.author_id
		FROM
			post
		WHERE
			post.id = $1
			AND NOT post.deleted
		`,
		postId,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			return false
		} else {
			panic(oops.New(err, "failed to get author of post when checking permissions"))
		}
	}

	return authorID != nil && *authorID == user.ID
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
		INSERT INTO post (postdate, thread_id, thread_type, current_id, author_id, project_id, reply_id, preview)
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

	// Track a project update
	updateEntries := []string{"all_last_updated"}
	switch threadType {
	case models.ThreadTypeForumPost:
		updateEntries = append(updateEntries, "forum_last_updated")
	case models.ThreadTypeProjectBlogPost, models.ThreadTypePersonalBlogPost:
		updateEntries = append(updateEntries, "blog_last_updated")
	}
	for i := range updateEntries {
		updateEntries[i] = fmt.Sprintf("%s = $2", updateEntries[i])
	}
	updates := strings.Join(updateEntries, ", ")

	_, err = tx.Exec(ctx,
		`
		UPDATE project
		SET `+updates+`
		WHERE
			id = $1
		`,
		projectId,
		time.Now(),
	)

	return
}

func DeletePost(
	ctx context.Context,
	tx pgx.Tx,
	threadId, postId int,
) (threadDeleted bool) {
	type threadInfo struct {
		FirstPostID int  `db:"first_id"`
		Deleted     bool `db:"deleted"`
	}
	info, err := db.QueryOne[threadInfo](ctx, tx,
		`
		SELECT $columns
		FROM
			thread
		WHERE
			thread.id = $1
		`,
		threadId,
	)
	if err != nil {
		panic(oops.New(err, "failed to fetch thread info"))
	}
	if info.Deleted {
		return true
	}
	isFirstPost := info.FirstPostID == postId

	if isFirstPost {
		// Just delete the whole thread and all its posts.
		_, err = tx.Exec(ctx,
			`
			UPDATE thread
			SET deleted = TRUE
			WHERE id = $1
			`,
			threadId,
		)
		_, err = tx.Exec(ctx,
			`
			UPDATE post
			SET deleted = TRUE
			WHERE thread_id = $1
			`,
			threadId,
		)

		return true
	}

	_, err = tx.Exec(ctx,
		`
		UPDATE post
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

const maxPostContentLength = 200000

func CreatePostVersion(ctx context.Context, tx pgx.Tx, postId int, unparsedContent string, ipString string, editReason string, editorId *int) (versionId int) {
	if len(unparsedContent) > maxPostContentLength {
		logging.ExtractLogger(ctx).Warn().
			Str("preview", unparsedContent[:400]).
			Msg("Somebody attempted to create an extremely long post. Content was truncated.")
		unparsedContent = unparsedContent[:maxPostContentLength-1]
	}

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
		INSERT INTO post_version (post_id, text_raw, text_parsed, ip, date, edit_reason, editor_id)
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
		UPDATE post
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

	// Update asset usage

	_, err = tx.Exec(ctx,
		`
		DELETE FROM post_asset_usage
		WHERE post_id = $1
		`,
		postId,
	)

	matches := hmnurl.RegexS3Asset.FindAllStringSubmatch(unparsedContent, -1)
	keyIdx := hmnurl.RegexS3Asset.SubexpIndex("key")

	var keys []string
	for _, match := range matches {
		key := match[keyIdx]
		keys = append(keys, key)
	}

	assetIDs, err := db.QueryScalar[uuid.UUID](ctx, tx,
		`
		SELECT id
		FROM asset
		WHERE s3_key = ANY($1)
		`,
		keys,
	)
	if err != nil {
		panic(oops.New(err, "failed to get assets matching keys"))
	}

	var values [][]interface{}

	for _, assetID := range assetIDs {
		values = append(values, []interface{}{postId, assetID})
	}

	_, err = tx.CopyFrom(ctx, pgx.Identifier{"post_asset_usage"}, []string{"post_id", "asset_id"}, pgx.CopyFromRows(values))
	if err != nil {
		panic(oops.New(err, "failed to insert post asset usage"))
	}

	return
}

var errThreadEmpty = errors.New("thread contained no non-deleted posts")

/*
Ensures that the first_id and last_id on the thread are still good.

Returns errThreadEmpty if the thread contains no visible posts any more.
You should probably mark the thread as deleted in this case.
*/
func FixThreadPostIds(ctx context.Context, conn db.ConnOrTx, threadId int) error {
	posts, err := db.Query[models.Post](ctx, conn,
		`
		SELECT $columns
		FROM post
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
	for _, post := range posts {
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

	_, err = conn.Exec(ctx,
		`
		UPDATE thread
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

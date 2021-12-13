package website

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4/pgxpool"
)

type AdminAtomFeedData struct {
	Title    string
	Subtitle string

	HomepageUrl string
	AtomFeedUrl string
	FeedUrl     string

	CopyrightStatement string
	SiteVersion        string
	Updated            time.Time
	FeedID             string

	Posts []templates.PostListItem
}

func AdminAtomFeed(c *RequestContext) ResponseData {
	creds := fmt.Sprintf("%s:%s", config.Config.Admin.AtomUsername, config.Config.Admin.AtomPassword)
	expectedAuth := fmt.Sprintf("Basic %s", base64.StdEncoding.EncodeToString([]byte(creds)))
	auth, hasAuth := c.Req.Header["Authorization"]
	if !hasAuth {
		res := ResponseData{
			StatusCode: http.StatusUnauthorized,
		}
		res.Header().Set("WWW-Authenticate", "Basic realm=\"Admin\"")
		return res
	} else if auth[0] != expectedAuth {
		return FourOhFour(c)
	}

	feedData := AdminAtomFeedData{
		HomepageUrl:        hmnurl.BuildHomepage(),
		CopyrightStatement: fmt.Sprintf("Copyright (C) 2014-%d Handmade.Network and its contributors", time.Now().Year()),
		SiteVersion:        "2.0",
		Title:              "Handmade Network Admin feed",
		Subtitle:           "Unapproved user posts",
		FeedID:             uuid.NewSHA1(uuid.NameSpaceURL, []byte(hmnurl.BuildAdminAtomFeed())).URN(),
		AtomFeedUrl:        hmnurl.BuildAdminAtomFeed(),
		FeedUrl:            hmnurl.BuildAdminApprovalQueue(),
	}

	unapprovedPosts, err := fetchUnapprovedPosts(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch unapproved posts"))
	}

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	for _, post := range unapprovedPosts {
		postItem := MakePostListItem(
			lineageBuilder,
			&post.Project,
			&post.Thread,
			&post.Post,
			&post.Author,
			false,
			true,
			c.Theme,
		)

		postItem.PostTypePrefix = fmt.Sprintf("ADMIN::UNAPPROVED: %s", postItem.PostTypePrefix)
		postItem.UUID = uuid.NewSHA1(uuid.NameSpaceURL, []byte(postItem.Url)).URN()
		postItem.LastEditDate = post.CurrentVersion.Date
		feedData.Posts = append(feedData.Posts, postItem)
	}
	if len(feedData.Posts) > 0 {
		feedData.Updated = feedData.Posts[0].Date
	} else {
		feedData.Updated = time.Now()
	}
	var res ResponseData
	res.MustWriteTemplate("admin_atom.xml", feedData, c.Perf)
	return res
}

const (
	ApprovalQueueActionApprove string = "approve"
	ApprovalQueueActionSpammer string = "spammer"
)

type postWithTitle struct {
	templates.Post
	Title string
}

type adminApprovalQueueData struct {
	templates.BaseData

	Posts          []postWithTitle
	SubmitUrl      string
	ApprovalAction string
	SpammerAction  string
}

func AdminApprovalQueue(c *RequestContext) ResponseData {
	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	posts, err := fetchUnapprovedPosts(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch unapproved posts"))
	}

	data := adminApprovalQueueData{
		BaseData:       getBaseDataAutocrumb(c, "Admin approval queue"),
		SubmitUrl:      hmnurl.BuildAdminApprovalQueue(),
		ApprovalAction: ApprovalQueueActionApprove,
		SpammerAction:  ApprovalQueueActionSpammer,
	}
	for _, p := range posts {
		post := templates.PostToTemplate(&p.Post, &p.Author, c.Theme)
		post.AddContentVersion(p.CurrentVersion, &p.Author) // NOTE(asaf): Don't care about editors here
		post.Url = UrlForGenericPost(hmndata.UrlContextForProject(&p.Project), &p.Thread, &p.Post, lineageBuilder)
		data.Posts = append(data.Posts, postWithTitle{
			Post:  post,
			Title: p.Thread.Title,
		})
	}

	var res ResponseData
	res.MustWriteTemplate("admin_approval_queue.html", data, c.Perf)
	return res
}

func AdminApprovalQueueSubmit(c *RequestContext) ResponseData {
	err := c.Req.ParseForm()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "failed to parse admin approval form"))
	}
	action := c.Req.Form.Get("action")
	userIdStr := c.Req.Form.Get("user_id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		return RejectRequest(c, "User id can't be parsed")
	}

	u, err := db.QueryOne(c.Context(), c.Conn, models.User{},
		`
		SELECT $columns FROM auth_user WHERE id = $1
		`,
		userId,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			return RejectRequest(c, "User not found")
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user"))
		}
	}
	user := u.(*models.User)

	whatHappened := ""
	if action == ApprovalQueueActionApprove {
		_, err := c.Conn.Exec(c.Context(),
			`
			UPDATE auth_user
			SET status = $1
			WHERE id = $2
			`,
			models.UserStatusApproved,
			user.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to set user to approved"))
		}
		whatHappened = fmt.Sprintf("%s approved successfully", user.Username)
	} else if action == ApprovalQueueActionSpammer {
		_, err := c.Conn.Exec(c.Context(),
			`
			UPDATE auth_user
			SET status = $1
			WHERE id = $2
			`,
			models.UserStatusBanned,
			user.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to set user to banned"))
		}
		err = auth.DeleteSessionForUser(c.Context(), c.Conn, user.Username)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to log out user"))
		}
		err = deleteAllPostsForUser(c.Context(), c.Conn, user.ID)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete spammer's posts"))
		}
		whatHappened = fmt.Sprintf("%s banned successfully", user.Username)
	} else {
		whatHappened = fmt.Sprintf("Unrecognized action: %s", action)
	}

	res := c.Redirect(hmnurl.BuildAdminApprovalQueue(), http.StatusSeeOther)
	res.AddFutureNotice("success", whatHappened)
	return res
}

type UnapprovedPost struct {
	Project        models.Project     `db:"project"`
	Thread         models.Thread      `db:"thread"`
	Post           models.Post        `db:"post"`
	CurrentVersion models.PostVersion `db:"ver"`
	Author         models.User        `db:"author"`
}

func fetchUnapprovedPosts(c *RequestContext) ([]*UnapprovedPost, error) {
	it, err := db.Query(c.Context(), c.Conn, UnapprovedPost{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			JOIN handmade_project AS project ON post.project_id = project.id
			JOIN handmade_thread AS thread ON post.thread_id = thread.id
			JOIN handmade_postversion AS ver ON ver.id = post.current_id
			JOIN auth_user AS author ON author.id = post.author_id
		WHERE
			NOT thread.deleted
			AND NOT post.deleted
			AND author.status = ANY($1)
		ORDER BY post.postdate DESC
		`,
		[]models.UserStatus{models.UserStatusConfirmed},
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch unapproved posts")
	}
	var res []*UnapprovedPost
	for _, iresult := range it.ToSlice() {
		res = append(res, iresult.(*UnapprovedPost))
	}
	return res, nil
}

func deleteAllPostsForUser(ctx context.Context, conn *pgxpool.Pool, userId int) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)
	type toDelete struct {
		ThreadID int `db:"thread.id"`
		PostID   int `db:"post.id"`
	}
	it, err := db.Query(ctx, tx, toDelete{},
		`
		SELECT $columns
		FROM
			handmade_post as post
			JOIN handmade_thread AS thread ON post.thread_id = thread.id
			JOIN auth_user AS author ON author.id = post.author_id
		WHERE author.id = $1
		`,
		userId,
	)

	if err != nil {
		return oops.New(err, "failed to fetch posts to delete for user")
	}

	for _, iResult := range it.ToSlice() {
		row := iResult.(*toDelete)
		hmndata.DeletePost(ctx, tx, row.ThreadID, row.PostID)
	}
	err = tx.Commit(ctx)
	if err != nil {
		return oops.New(err, "failed to commit transaction")
	}
	return nil
}

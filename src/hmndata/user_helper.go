package hmndata

import (
	"context"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
)

type UsersQuery struct {
	// Ignored when using FetchUser
	UserIDs   []int    // if empty, all users
	Usernames []string // if empty, all users

	// Flags to modify behavior
	AnyStatus bool // Bypasses shadowban system
}

/*
Fetches users and related models from the database according to all the given
query params. For the most correct results, provide as much information as you have
on hand.
*/
func FetchUsers(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q UsersQuery,
) ([]*models.User, error) {
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch users")
	defer perf.EndBlock()

	var currentUserID *int
	if currentUser != nil {
		currentUserID = &currentUser.ID
	}

	for i := range q.Usernames {
		q.Usernames[i] = strings.ToLower(q.Usernames[i])
	}

	type userRow struct {
		User        models.User   `db:"hmn_user"`
		AvatarAsset *models.Asset `db:"avatar"`
	}

	var qb db.QueryBuilder
	qb.Add(`
		SELECT $columns
		FROM
			hmn_user
			LEFT JOIN asset AS avatar ON avatar.id = hmn_user.avatar_asset_id
		WHERE
			TRUE
	`)
	if len(q.UserIDs) > 0 {
		qb.Add(`AND hmn_user.id = ANY($?)`, q.UserIDs)
	}
	if len(q.Usernames) > 0 {
		qb.Add(`AND LOWER(hmn_user.username) = ANY($?)`, q.Usernames)
	}
	if !q.AnyStatus {
		if currentUser == nil {
			qb.Add(`AND hmn_user.status = $?`, models.UserStatusApproved)
		} else if !currentUser.IsStaff {
			qb.Add(
				`
				AND (
					hmn_user.status = $? -- user is Approved
					OR hmn_user.id = $? -- getting self
				)
				`,
				models.UserStatusApproved,
				currentUserID,
			)
		}
	}

	userRows, err := db.Query[userRow](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch users")
	}

	result := make([]*models.User, len(userRows))
	for i, row := range userRows {
		user := row.User
		user.AvatarAsset = row.AvatarAsset
		result[i] = &user
	}

	return result, nil
}

/*
Fetches a single user and related data. A wrapper around FetchUsers.
As with FetchUsers, provide as much information as you know to get the
most correct results.

Returns db.NotFound if no result is found.
*/
func FetchUser(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	userID int,
	q UsersQuery,
) (*models.User, error) {
	q.UserIDs = []int{userID}

	res, err := FetchUsers(ctx, dbConn, currentUser, q)
	if err != nil {
		return nil, oops.New(err, "failed to fetch user")
	}

	if len(res) == 0 {
		return nil, db.NotFound
	}

	return res[0], nil
}

/*
Fetches a single user and related data. A wrapper around FetchUsers.
As with FetchUsers, provide as much information as you know to get the
most correct results.

Returns db.NotFound if no result is found.
*/
func FetchUserByUsername(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	username string,
	q UsersQuery,
) (*models.User, error) {
	q.Usernames = []string{username}

	res, err := FetchUsers(ctx, dbConn, currentUser, q)
	if err != nil {
		return nil, oops.New(err, "failed to fetch user")
	}

	if len(res) == 0 {
		return nil, db.NotFound
	}

	return res[0], nil
}

// NOTE(ben): Someday we can add CountUsers...I don't have a need for it right now.

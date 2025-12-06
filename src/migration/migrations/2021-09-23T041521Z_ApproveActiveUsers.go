package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(ApproveActiveUsers{})
}

type ApproveActiveUsers struct{}

func (m ApproveActiveUsers) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 9, 23, 4, 15, 21, 0, time.UTC))
}

func (m ApproveActiveUsers) Name() string {
	return "ApproveActiveUsers"
}

func (m ApproveActiveUsers) Description() string {
	return "Give legit users the Approved status"
}

func (m ApproveActiveUsers) Up(ctx context.Context, tx pgx.Tx) error {
	/*
		See models/user.go.
		The old statuses were:
			2 = Active
			3 = Banned
		The new statuses are:
			2 = Confirmed (valid email)
			3 = Approved (allowed to post)
			4 = Banned
	*/

	_, err := tx.Exec(ctx, `
		UPDATE auth_user
		SET status = 4
		WHERE status = 3
	`)
	if err != nil {
		return oops.New(err, "failed to update status of banned users")
	}

	_, err = tx.Exec(ctx, `
		UPDATE auth_user
		SET status = 3
		WHERE
			status = 2
			AND id IN (
				SELECT author_id
				FROM handmade_post
				WHERE author_id IS NOT NULL
			)
	`)
	if err != nil {
		return oops.New(err, "failed to update user statuses")
	}

	return nil
}

func (m ApproveActiveUsers) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		UPDATE auth_user
		SET status = 2
		WHERE status = 3
	`)
	if err != nil {
		return oops.New(err, "failed to revert approved users back to confirmed")
	}

	_, err = tx.Exec(ctx, `
		UPDATE auth_user
		SET status = 3
		WHERE status = 4
	`)
	if err != nil {
		return oops.New(err, "failed to update status of banned users")
	}

	return nil
}

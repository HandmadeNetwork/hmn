package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(DropPostParentID{})
}

type DropPostParentID struct{}

func (m DropPostParentID) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 30, 16, 42, 40, 0, time.UTC))
}

func (m DropPostParentID) Name() string {
	return "DropPostParentID"
}

func (m DropPostParentID) Description() string {
	return "Drop the parent_id field from posts"
}

func (m DropPostParentID) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_post
			DROP parent_id;
	`)
	if err != nil {
		return oops.New(err, "failed to drop parent_id field")
	}

	return nil
}

func (m DropPostParentID) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

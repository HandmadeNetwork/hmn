package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(AddSubforumSort{})
}

type AddSubforumSort struct{}

func (m AddSubforumSort) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 9, 23, 6, 12, 11, 0, time.UTC))
}

func (m AddSubforumSort) Name() string {
	return "AddSubforumSort"
}

func (m AddSubforumSort) Description() string {
	return "Adds a sort field to subforums."
}

func (m AddSubforumSort) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_subforum
			ADD sort INTEGER NOT NULL DEFAULT 10;
	`)
	if err != nil {
		return oops.New(err, "failed to add sort key")
	}

	return nil
}

func (m AddSubforumSort) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_subforum
			DROP sort;
	`)
	if err != nil {
		return oops.New(err, "failed to drop sort key")
	}

	return nil
}

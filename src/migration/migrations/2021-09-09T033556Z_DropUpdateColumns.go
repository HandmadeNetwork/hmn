package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(DropUpdateColumns{})
}

type DropUpdateColumns struct{}

func (m DropUpdateColumns) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 9, 9, 3, 35, 56, 0, time.UTC))
}

func (m DropUpdateColumns) Name() string {
	return "DropUpdateColumns"
}

func (m DropUpdateColumns) Description() string {
	return "Drop old columns related to update times"
}

func (m DropUpdateColumns) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_project
			DROP profile_last_updated,
			DROP static_last_updated;
	`)
	if err != nil {
		return oops.New(err, "failed to drop update columns")
	}

	return nil
}

func (m DropUpdateColumns) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_project
			ADD profile_last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch',
			ADD static_last_updated TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch';
	`)
	if err != nil {
		return oops.New(err, "failed to add old update columns")
	}

	return nil
}

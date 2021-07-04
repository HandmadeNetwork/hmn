package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(DropThreadFields{})
}

type DropThreadFields struct{}

func (m DropThreadFields) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 4, 21, 36, 58, 0, time.UTC))
}

func (m DropThreadFields) Name() string {
	return "DropThreadFields"
}

func (m DropThreadFields) Description() string {
	return "Drop unnecessary thread fields"
}

func (m DropThreadFields) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_thread
			DROP hits,
			DROP reply_count;
	`)
	if err != nil {
		return oops.New(err, "failed to drop thread fields")
	}

	return nil
}

func (m DropThreadFields) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

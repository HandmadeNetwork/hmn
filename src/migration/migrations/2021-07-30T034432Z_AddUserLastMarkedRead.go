package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(AddUserLastMarkedRead{})
}

type AddUserLastMarkedRead struct{}

func (m AddUserLastMarkedRead) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 30, 3, 44, 32, 0, time.UTC))
}

func (m AddUserLastMarkedRead) Name() string {
	return "AddUserLastMarkedRead"
}

func (m AddUserLastMarkedRead) Description() string {
	return "Add a field to users that tracks when they last read EVERYTHING."
}

func (m AddUserLastMarkedRead) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE auth_user
			ADD marked_all_read_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch';
	`)
	if err != nil {
		return oops.New(err, "failed to insert new column")
	}

	return nil
}

func (m AddUserLastMarkedRead) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

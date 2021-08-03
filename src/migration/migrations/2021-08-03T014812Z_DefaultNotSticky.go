package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(DefaultNotSticky{})
}

type DefaultNotSticky struct{}

func (m DefaultNotSticky) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 3, 1, 48, 12, 0, time.UTC))
}

func (m DefaultNotSticky) Name() string {
	return "DefaultNotSticky"
}

func (m DefaultNotSticky) Description() string {
	return "Make sticky default to false"
}

func (m DefaultNotSticky) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_thread
			ALTER sticky SET DEFAULT FALSE;
	`)
	if err != nil {
		return oops.New(err, "failed to set default")
	}

	return nil
}

func (m DefaultNotSticky) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(RemoveProjectNulls{})
}

type RemoveProjectNulls struct{}

func (m RemoveProjectNulls) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 5, 6, 3, 13, 28, 0, time.UTC))
}

func (m RemoveProjectNulls) Name() string {
	return "RemoveProjectNulls"
}

func (m RemoveProjectNulls) Description() string {
	return "Make project fields non-nullable"
}

func (m RemoveProjectNulls) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_project
			ALTER slug SET NOT NULL,
			ALTER name SET NOT NULL,
			ALTER blurb SET NOT NULL,
			ALTER description SET NOT NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to make project fields non-null")
	}

	return nil
}

func (m RemoveProjectNulls) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

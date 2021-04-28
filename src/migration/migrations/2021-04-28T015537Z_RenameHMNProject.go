package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(RenameHMNProject{})
}

type RenameHMNProject struct{}

func (m RenameHMNProject) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 28, 1, 55, 37, 0, time.UTC))
}

func (m RenameHMNProject) Name() string {
	return "RenameHMNProject"
}

func (m RenameHMNProject) Description() string {
	return "Rename the special HMN project"
}

func (m RenameHMNProject) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `UPDATE handmade_project SET name = 'Handmade Network' WHERE id = 1`)
	if err != nil {
		return oops.New(err, "failed to rename project")
	}

	return nil
}

func (m RenameHMNProject) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

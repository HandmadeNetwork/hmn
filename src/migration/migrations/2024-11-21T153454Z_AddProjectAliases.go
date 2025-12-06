package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddProjectAliases{})
}

type AddProjectAliases struct{}

func (m AddProjectAliases) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 11, 21, 15, 34, 54, 0, time.UTC))
}

func (m AddProjectAliases) Name() string {
	return "AddProjectAliases"
}

func (m AddProjectAliases) Description() string {
	return "Add aliases to projects so we can resolve multiple subdomains"
}

func (m AddProjectAliases) Up(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE project
			ADD COLUMN slug_aliases VARCHAR(30)[] NOT NULL DEFAULT '{}';
		`,
	))
	return nil
}

func (m AddProjectAliases) Down(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE project
			DROP COLUMN slug_aliases;
		`,
	))
	return nil
}

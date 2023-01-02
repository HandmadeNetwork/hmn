package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddPersistentVars{})
}

type AddPersistentVars struct{}

func (m AddPersistentVars) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 5, 26, 14, 45, 17, 0, time.UTC))
}

func (m AddPersistentVars) Name() string {
	return "AddPersistentVars"
}

func (m AddPersistentVars) Description() string {
	return "Create table for persistent_vars"
}

func (m AddPersistentVars) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE persistent_var (
			name VARCHAR(255) NOT NULL,
			value TEXT NOT NULL
		);
		CREATE UNIQUE INDEX persistent_var_name ON persistent_var (name);
		`,
	)
	if err != nil {
		return oops.New(err, "failed to create persistent_var table")
	}
	return nil
}

func (m AddPersistentVars) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP INDEX persistent_var_name;
		DROP TABLE persistent_var;
		`,
	)
	if err != nil {
		return oops.New(err, "failed to drop persistent_var table")
	}
	return nil
}

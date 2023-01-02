package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddSessionTable{})
}

type AddSessionTable struct{}

func (m AddSessionTable) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 3, 26, 3, 38, 34, 0, time.UTC))
}

func (m AddSessionTable) Name() string {
	return "AddSessionTable"
}

func (m AddSessionTable) Description() string {
	return "Adds a session table to replace the Django session table"
}

func (m AddSessionTable) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		CREATE TABLE sessions (
			id VARCHAR(40) PRIMARY KEY,
			username VARCHAR(150) NOT NULL,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL
		);
	`)
	return err
}

func (m AddSessionTable) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		DROP TABLE sessions;
	`)
	return err
}

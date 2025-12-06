package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddPendingSignups{})
}

type AddPendingSignups struct{}

func (m AddPendingSignups) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2023, 5, 4, 2, 47, 12, 0, time.UTC))
}

func (m AddPendingSignups) Name() string {
	return "AddPendingSignups"
}

func (m AddPendingSignups) Description() string {
	return "Adds the pending login table"
}

func (m AddPendingSignups) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE pending_login (
			id VARCHAR(40) NOT NULL PRIMARY KEY,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
			destination_url VARCHAR(999) NOT NULL
		)
		`,
	)
	if err != nil {
		return err
	}

	return nil
}

func (m AddPendingSignups) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `DROP TABLE pending_login`)
	if err != nil {
		return err
	}

	return nil
}

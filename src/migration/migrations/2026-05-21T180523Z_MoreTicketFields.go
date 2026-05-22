package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(MoreTicketFields{})
}

type MoreTicketFields struct{}

func (m MoreTicketFields) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 5, 21, 18, 5, 23, 0, time.UTC))
}

func (m MoreTicketFields) Name() string {
	return "MoreTicketFields"
}

func (m MoreTicketFields) Description() string {
	return "Add accommodation and shirtsize columns"
}

func (m MoreTicketFields) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE ticket
		ADD COLUMN accommodations TEXT NOT NULL DEFAULT '';
		`,
	)
	return err
}

func (m MoreTicketFields) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE ticket
		DROP COLUMN accommodations;
		`,
	)
	return err
}

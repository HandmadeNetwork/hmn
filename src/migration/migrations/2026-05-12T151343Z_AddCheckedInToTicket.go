package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddCheckedInToTicket{})
}

type AddCheckedInToTicket struct{}

func (m AddCheckedInToTicket) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 5, 12, 15, 13, 43, 0, time.UTC))
}

func (m AddCheckedInToTicket) Name() string {
	return "AddCheckedInToTicket"
}

func (m AddCheckedInToTicket) Description() string {
	return "Add checked-in to ticket"
}

func (m AddCheckedInToTicket) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE ticket
		ADD COLUMN checked_in BOOLEAN NOT NULL DEFAULT FALSE;
		`,
	)
	return err
}

func (m AddCheckedInToTicket) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE ticket
		DROP COLUMN checked_in;
		`,
	)
	return err
}

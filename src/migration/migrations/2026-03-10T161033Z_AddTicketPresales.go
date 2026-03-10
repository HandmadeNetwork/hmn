package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddTicketPresales{})
}

type AddTicketPresales struct{}

func (m AddTicketPresales) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 3, 10, 16, 10, 33, 0, time.UTC))
}

func (m AddTicketPresales) Name() string {
	return "AddTicketPresales"
}

func (m AddTicketPresales) Description() string {
	return "Adds a presale column to ticket event metadata"
}

func (m AddTicketPresales) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE ticket_metadata
			ADD COLUMN presale BOOL NOT NULL DEFAULT FALSE;
		`,
	)
	return err
}

func (m AddTicketPresales) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE ticket_metadata
			DROP COLUMN presale;
		`,
	)
	return err
}

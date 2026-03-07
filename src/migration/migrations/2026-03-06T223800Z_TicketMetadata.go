package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(TicketMetadata{})
}

type TicketMetadata struct{}

func (m TicketMetadata) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 3, 6, 22, 38, 0, 0, time.UTC))
}

func (m TicketMetadata) Name() string {
	return "TicketMetadata"
}

func (m TicketMetadata) Description() string {
	return "Add the ticket metadata table"
}

func (m TicketMetadata) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE ticket_metadata (
			slug VARCHAR(64) UNIQUE NOT NULL,
			max_tickets INT NOT NULL DEFAULT 0,
			max_reserved INT NOT NULL DEFAULT 0,
			price_amount VARCHAR(10) NOT NULL DEFAULT '',
			price_currency VARCHAR(10) NOT NULL DEFAULT 'USD'
		)
		`,
	)
	return err
}

func (m TicketMetadata) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP TABLE ticket_metadata
		`,
	)

	return err
}

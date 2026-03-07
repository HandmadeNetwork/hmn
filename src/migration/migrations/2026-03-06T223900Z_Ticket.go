package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(Ticket{})
}

type Ticket struct{}

func (m Ticket) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 3, 6, 22, 39, 0, 0, time.UTC))
}

func (m Ticket) Name() string {
	return "Ticket"
}

func (m Ticket) Description() string {
	return "Add ticket table"
}

func (m Ticket) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE ticket (
			id UUID NOT NULL,
			event_slug VARCHAR(64) NOT NULL,

			-- We don't want to delete tickets that have already been paid. We expect that it will be
			-- quite rare for a user to request deletion after ever paying for a ticket to an HMN event,
			-- and if they do, we can handle it manually by deleting the ticket.
			user_id INT REFERENCES hmn_user(id) ON DELETE RESTRICT,

			name TEXT NOT NULL,
			email TEXT NOT NULL,

			pending BOOLEAN NOT NULL DEFAULT FALSE,
			reserved BOOLEAN NOT NULL DEFAULT FALSE,

			purchase_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

			stripe_cs_id VARCHAR(2048) NOT NULL DEFAULT '',
			stripe_pi_id VARCHAR(2048) NOT NULL DEFAULT '',

			price_amount VARCHAR(10) NOT NULL DEFAULT '',
			price_currency VARCHAR(10) NOT NULL DEFAULT '',
			notes TEXT NOT NULL DEFAULT ''
		)
		`,
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`
		CREATE INDEX ticket_slug ON ticket (event_slug);
		CREATE INDEX ticket_reserved ON ticket (reserved);
		`,
	)
	return err
}

func (m Ticket) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP INDEX ticket_slug;
		DROP INDEX ticket_reserved;
		DROP TABLE ticket;
		`,
	)
	return err
}

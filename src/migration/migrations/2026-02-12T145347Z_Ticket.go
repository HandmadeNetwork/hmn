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
	return types.MigrationVersion(time.Date(2026, 2, 12, 14, 53, 47, 0, time.UTC))
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
			id UUID not null,
			event_slug VARCHAR(64) NOT NULL,
			owner_user_id INT REFERENCES hmn_user (id),
			name TEXT,
			email TEXT,
			reserved BOOLEAN NOT NULL DEFAULT FALSE,
			allocation_date TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			price_amount VARCHAR(10) NOT NULL,
			price_currency VARCHAR(10) NOT NULL,
			notes TEXT
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

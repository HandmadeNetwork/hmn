package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddStripeMembershipEventCursor{})
}

type AddStripeMembershipEventCursor struct{}

func (m AddStripeMembershipEventCursor) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 6, 2, 9, 35, 0, 0, time.UTC))
}

func (m AddStripeMembershipEventCursor) Name() string {
	return "AddStripeMembershipEventCursor"
}

func (m AddStripeMembershipEventCursor) Description() string {
	return "Add per-customer Stripe membership event ordering cursor"
}

func (m AddStripeMembershipEventCursor) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		CREATE TABLE stripe_membership_event_cursor (
			customer_id TEXT PRIMARY KEY,
			last_event_created BIGINT NOT NULL,
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func (m AddStripeMembershipEventCursor) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		DROP TABLE stripe_membership_event_cursor
	`)
	return err
}

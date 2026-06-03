package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddStripeWebhookEventLedger{})
}

type AddStripeWebhookEventLedger struct{}

func (m AddStripeWebhookEventLedger) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 6, 1, 5, 56, 0, 0, time.UTC))
}

func (m AddStripeWebhookEventLedger) Name() string {
	return "AddStripeWebhookEventLedger"
}

func (m AddStripeWebhookEventLedger) Description() string {
	return "Add Stripe webhook event idempotency ledger table"
}

func (m AddStripeWebhookEventLedger) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		CREATE TABLE stripe_webhook_event (
			event_id TEXT PRIMARY KEY,
			event_type TEXT NOT NULL,
			status TEXT NOT NULL DEFAULT 'processing',
			last_error TEXT,
			received_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			processed_at TIMESTAMP WITH TIME ZONE
		)
	`)
	return err
}

func (m AddStripeWebhookEventLedger) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		DROP TABLE stripe_webhook_event
	`)
	return err
}

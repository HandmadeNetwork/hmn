package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddStripeToUser{})
}

type AddStripeToUser struct{}

func (m AddStripeToUser) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 1, 8, 5, 23, 37, 0, time.UTC))
}

func (m AddStripeToUser) Name() string {
	return "AddStripeToUser"
}

func (m AddStripeToUser) Description() string {
	return "Add Stripe customer and subscription IDs to user table"
}

func (m AddStripeToUser) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user 
		ADD COLUMN stripe_customer_id TEXT,
		ADD COLUMN stripe_subscription_id TEXT
	`)
	return err
}

func (m AddStripeToUser) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user 
		DROP COLUMN stripe_customer_id,
		DROP COLUMN stripe_subscription_id
	`)
	return err
}

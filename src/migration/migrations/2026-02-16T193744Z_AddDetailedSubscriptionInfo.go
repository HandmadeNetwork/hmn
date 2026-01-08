package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddDetailedSubscriptionInfo{})
}

type AddDetailedSubscriptionInfo struct{}

func (m AddDetailedSubscriptionInfo) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 2, 16, 19, 37, 44, 0, time.UTC))
}

func (m AddDetailedSubscriptionInfo) Name() string {
	return "AddDetailedSubscriptionInfo"
}

func (m AddDetailedSubscriptionInfo) Description() string {
	return "Add detailed subscription fields to user table and create user_payment table"
}

func (m AddDetailedSubscriptionInfo) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user 
		ADD COLUMN subscription_status TEXT,
		ADD COLUMN current_period_end TIMESTAMP WITH TIME ZONE,
		ADD COLUMN cancel_at_period_end BOOLEAN NOT NULL DEFAULT false;

		CREATE TABLE user_payment (
			id SERIAL PRIMARY KEY,
			user_id INTEGER NOT NULL REFERENCES hmn_user(id) ON DELETE CASCADE,
			stripe_invoice_id TEXT UNIQUE,
			amount_cents INTEGER NOT NULL,
			currency TEXT NOT NULL,
			payment_method_type TEXT,
			card_last4 TEXT,
			card_brand TEXT,
			paid_at TIMESTAMP WITH TIME ZONE NOT NULL
		);

		CREATE INDEX idx_user_payment_user_id ON user_payment(user_id);
	`)
	return err
}

func (m AddDetailedSubscriptionInfo) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		DROP TABLE user_payment;

		ALTER TABLE hmn_user 
		DROP COLUMN subscription_status,
		DROP COLUMN current_period_end,
		DROP COLUMN cancel_at_period_end;
	`)
	return err
}

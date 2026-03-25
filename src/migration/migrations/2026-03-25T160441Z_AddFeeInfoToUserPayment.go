package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddFeeInfoToUserPayment{})
}

type AddFeeInfoToUserPayment struct{}

func (m AddFeeInfoToUserPayment) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 3, 25, 16, 4, 41, 0, time.UTC))
}

func (m AddFeeInfoToUserPayment) Name() string {
	return "AddFeeInfoToUserPayment"
}

func (m AddFeeInfoToUserPayment) Description() string {
	return "Add stripe_fee_cents and net_amount_cents to user_payment table"
}

func (m AddFeeInfoToUserPayment) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE user_payment 
		ADD COLUMN stripe_fee_cents INTEGER,
		ADD COLUMN net_amount_cents INTEGER;
	`)
	return err
}

func (m AddFeeInfoToUserPayment) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE user_payment 
		DROP COLUMN stripe_fee_cents,
		DROP COLUMN net_amount_cents;
	`)
	return err
}

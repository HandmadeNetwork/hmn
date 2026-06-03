package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddSubscriptionGracePeriod{})
}

type AddSubscriptionGracePeriod struct{}

func (m AddSubscriptionGracePeriod) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 5, 24, 17, 0, 0, 0, time.UTC))
}

func (m AddSubscriptionGracePeriod) Name() string {
	return "AddSubscriptionGracePeriod"
}

func (m AddSubscriptionGracePeriod) Description() string {
	return "Add grace period tracking columns to hmn_user"
}

func (m AddSubscriptionGracePeriod) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user
		ADD COLUMN grace_period_started_at TIMESTAMP WITH TIME ZONE,
		ADD COLUMN grace_period_ends_at TIMESTAMP WITH TIME ZONE,
		ADD COLUMN grace_available BOOLEAN NOT NULL DEFAULT true;
	`)
	return err
}

func (m AddSubscriptionGracePeriod) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user
		DROP COLUMN grace_period_started_at,
		DROP COLUMN grace_period_ends_at,
		DROP COLUMN grace_available;
	`)
	return err
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddIsSubscribedToUser{})
}

type AddIsSubscribedToUser struct{}

func (m AddIsSubscribedToUser) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 1, 7, 23, 32, 36, 0, time.UTC))
}

func (m AddIsSubscribedToUser) Name() string {
	return "AddIsSubscribedToUser"
}

func (m AddIsSubscribedToUser) Description() string {
	return "Add is_subscribed column to hmn_user table"
}

func (m AddIsSubscribedToUser) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE hmn_user
			ADD COLUMN is_subscribed BOOLEAN NOT NULL DEFAULT false;
		`,
	)
	return err
}

func (m AddIsSubscribedToUser) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE hmn_user
			DROP COLUMN is_subscribed;
		`,
	)
	return err
}

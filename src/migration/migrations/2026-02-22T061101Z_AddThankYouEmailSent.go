package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddThankYouEmailSent{})
}

type AddThankYouEmailSent struct{}

func (m AddThankYouEmailSent) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 2, 22, 6, 11, 1, 0, time.UTC))
}

func (m AddThankYouEmailSent) Name() string {
	return "AddThankYouEmailSent"
}

func (m AddThankYouEmailSent) Description() string {
	return "Add thank_you_email_sent column to hmn_user table"
}

func (m AddThankYouEmailSent) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user
		ADD COLUMN thank_you_email_sent BOOLEAN NOT NULL DEFAULT false;
	`)
	return err
}

func (m AddThankYouEmailSent) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE hmn_user
		DROP COLUMN thank_you_email_sent;
	`)
	return err
}

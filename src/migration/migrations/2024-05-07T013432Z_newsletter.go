package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(newsletter{})
}

type newsletter struct{}

func (m newsletter) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 5, 7, 1, 34, 32, 0, time.UTC))
}

func (m newsletter) Name() string {
	return "newsletter"
}

func (m newsletter) Description() string {
	return "Adds the newsletter signup"
}

func (m newsletter) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE newsletter_emails (
			email VARCHAR(255) NOT NULL PRIMARY KEY
		);
		`,
	)
	return err
}

func (m newsletter) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP TABLE newsletter_emails;
		`,
	)
	return err
}

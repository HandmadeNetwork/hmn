package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(NonNullProjectScreenshots{})
}

type NonNullProjectScreenshots struct{}

func (m NonNullProjectScreenshots) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 7, 25, 14, 11, 6, 0, time.UTC))
}

func (m NonNullProjectScreenshots) Name() string {
	return "NonNullProjectScreenshots"
}

func (m NonNullProjectScreenshots) Description() string {
	return "Make the asset column on project screenshots NOT NULL"
}

func (m NonNullProjectScreenshots) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project_screenshot
			ALTER COLUMN asset_id SET NOT NULL;
		`,
	)
	return err
}

func (m NonNullProjectScreenshots) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project_screenshot
			ALTER COLUMN asset_id DROP NOT NULL;
		`,
	)
	return err
}

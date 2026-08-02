package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(ProjectScreenshotSort{})
}

type ProjectScreenshotSort struct{}

func (m ProjectScreenshotSort) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 8, 1, 13, 16, 48, 0, time.UTC))
}

func (m ProjectScreenshotSort) Name() string {
	return "ProjectScreenshotSort"
}

func (m ProjectScreenshotSort) Description() string {
	return "Add a sort field to project screenshots"
}

func (m ProjectScreenshotSort) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE project_screenshot
			ADD COLUMN sort INTEGER NOT NULL DEFAULT 0,
			ADD CONSTRAINT project_screenshot_project_asset_uniq UNIQUE (project_id, asset_id);
	`)
	return err
}

func (m ProjectScreenshotSort) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE project_screenshot
			DROP CONSTRAINT project_screenshot_project_asset_uniq,
			DROP COLUMN sort;
	`)
	return err
}

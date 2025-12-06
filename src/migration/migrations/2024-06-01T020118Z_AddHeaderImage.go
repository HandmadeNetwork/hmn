package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddHeaderImage{})
}

type AddHeaderImage struct{}

func (m AddHeaderImage) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 6, 1, 2, 1, 18, 0, time.UTC))
}

func (m AddHeaderImage) Name() string {
	return "AddHeaderImage"
}

func (m AddHeaderImage) Description() string {
	return "Adds a header image to projects"
}

func (m AddHeaderImage) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project
			ADD COLUMN header_asset_id UUID REFERENCES asset (id) ON DELETE SET NULL;
		`,
	)
	return err
}

func (m AddHeaderImage) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project
			DROP COLUMN header_asset_id;
		`,
	)
	return err
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(AddLogoAssetsToProjects{})
}

type AddLogoAssetsToProjects struct{}

func (m AddLogoAssetsToProjects) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 11, 28, 16, 23, 0, 0, time.UTC))
}

func (m AddLogoAssetsToProjects) Name() string {
	return "AddLogoAssetsToProjects"
}

func (m AddLogoAssetsToProjects) Description() string {
	return "Add optional asset references for project logos"
}

func (m AddLogoAssetsToProjects) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_project
			ADD COLUMN logodark_asset_id UUID REFERENCES handmade_asset (id) ON DELETE SET NULL,
			ADD COLUMN logolight_asset_id UUID REFERENCES handmade_asset (id) ON DELETE SET NULL;
		`,
	)
	return err
}

func (m AddLogoAssetsToProjects) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_project
			DROP COLUMN logodark_asset_id,
			DROP COLUMN logolight_asset_id;
		`,
	)
	return err
}

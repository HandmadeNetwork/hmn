package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(RemoveDarkLogos{})
}

type RemoveDarkLogos struct{}

func (m RemoveDarkLogos) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2025, 5, 24, 18, 1, 27, 0, time.UTC))
}

func (m RemoveDarkLogos) Name() string {
	return "RemoveDarkLogos"
}

func (m RemoveDarkLogos) Description() string {
	return "Remove logos that are specific to dark mode"
}

func (m RemoveDarkLogos) Up(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE project
			RENAME logolight_asset_id TO logo_asset_id;
		ALTER TABLE project
			DROP logodark_asset_id;
		`,
	))
	return nil
}

func (m RemoveDarkLogos) Down(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE project
			RENAME logo_asset_id TO logolight_asset_id;
		ALTER TABLE project
			ADD COLUMN logodark_asset_id UUID REFERENCES handmade_asset (id) ON DELETE SET NULL,
		`,
	))
	panic("noerp")
}

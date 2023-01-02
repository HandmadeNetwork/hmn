package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(UserAvatarAssetId{})
}

type UserAvatarAssetId struct{}

func (m UserAvatarAssetId) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 12, 26, 10, 16, 33, 0, time.UTC))
}

func (m UserAvatarAssetId) Name() string {
	return "UserAvatarAssetId"
}

func (m UserAvatarAssetId) Description() string {
	return "Add avatar_asset_id to users"
}

func (m UserAvatarAssetId) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE auth_user
			ADD COLUMN avatar_asset_id UUID REFERENCES handmade_asset (id) ON DELETE SET NULL;
		`,
	)
	return err
}

func (m UserAvatarAssetId) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE auth_user
			DROP COLUMN avatar_asset_id;
		`,
	)
	return err
}

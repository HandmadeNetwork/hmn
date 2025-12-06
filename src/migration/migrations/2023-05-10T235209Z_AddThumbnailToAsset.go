package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddThumbnailToAsset{})
}

type AddThumbnailToAsset struct{}

func (m AddThumbnailToAsset) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2023, 5, 10, 23, 52, 9, 0, time.UTC))
}

func (m AddThumbnailToAsset) Name() string {
	return "AddThumbnailToAsset"
}

func (m AddThumbnailToAsset) Description() string {
	return "Adds thumbnail S3 key to assets"
}

func (m AddThumbnailToAsset) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE asset
		ADD COLUMN thumbnail_s3_key VARCHAR(2000);
		`,
	)
	return err
}

func (m AddThumbnailToAsset) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE asset
		DROP COLUMN thumbnail_s3_key;
		`,
	)
	return err
}

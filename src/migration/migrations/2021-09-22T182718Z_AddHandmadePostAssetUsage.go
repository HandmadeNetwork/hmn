package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddHandmadePostAssetUsage{})
}

type AddHandmadePostAssetUsage struct{}

func (m AddHandmadePostAssetUsage) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 9, 22, 18, 27, 18, 0, time.UTC))
}

func (m AddHandmadePostAssetUsage) Name() string {
	return "AddHandmadePostAssetUsage"
}

func (m AddHandmadePostAssetUsage) Description() string {
	return "Add table for tracking asset usage in posts, and a unique index on handmade_asset.s3_key"
}

func (m AddHandmadePostAssetUsage) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE handmade_post_asset_usage (
			post_id INTEGER NOT NULL REFERENCES handmade_post(id) ON DELETE CASCADE,
			asset_id UUID NOT NULL REFERENCES handmade_asset(id) ON DELETE CASCADE,
			CONSTRAINT handmade_post_asset_usage_unique UNIQUE(post_id, asset_id)
		);

		CREATE INDEX handmade_post_asset_usage_post_id ON handmade_post_asset_usage(post_id);
		CREATE INDEX handmade_post_asset_usage_asset_id ON handmade_post_asset_usage(asset_id);

		ALTER TABLE handmade_asset
			ADD CONSTRAINT handmade_asset_s3_key UNIQUE(s3_key);
		`,
	)
	if err != nil {
		return oops.New(err, "failed to add table and indexes")
	}
	return nil
}

func (m AddHandmadePostAssetUsage) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP INDEX handmade_post_asset_usage_post_id;
		DROP INDEX handmade_post_asset_usage_asset_id;
		DROP TABLE handmade_post_asset_usage;
		`,
	)
	if err != nil {
		return oops.New(err, "failed to drop table and indexes")
	}
	return nil
}

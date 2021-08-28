package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(RemoveFeatureFlags{})
}

type RemoveFeatureFlags struct{}

func (m RemoveFeatureFlags) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 28, 12, 49, 59, 0, time.UTC))
}

func (m RemoveFeatureFlags) Name() string {
	return "RemoveFeatureFlags"
}

func (m RemoveFeatureFlags) Description() string {
	return "Convert project feature flags to booleans"
}

func (m RemoveFeatureFlags) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_project
			DROP wiki_flags,
			DROP wiki_last_updated,
			DROP static_flags,
			DROP annotation_flags
	`)
	if err != nil {
		return oops.New(err, "failed to drop stupid old columns")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project
			ALTER forum_flags TYPE BOOLEAN USING forum_flags > 0,
			ALTER blog_flags TYPE BOOLEAN USING blog_flags > 0,
			ALTER library_flags TYPE BOOLEAN USING library_flags > 0;
		
		ALTER TABLE handmade_project RENAME forum_flags TO forum_enabled;
		ALTER TABLE handmade_project RENAME blog_flags TO blog_enabled;
		ALTER TABLE handmade_project RENAME library_flags TO library_enabled;
	`)
	if err != nil {
		return oops.New(err, "failed to convert flags to bools")
	}

	return nil
}

func (m RemoveFeatureFlags) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

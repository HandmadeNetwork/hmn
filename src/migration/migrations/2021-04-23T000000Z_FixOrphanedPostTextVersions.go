package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(FixOrphanedPostTextVersions{})
}

type FixOrphanedPostTextVersions struct{}

func (m FixOrphanedPostTextVersions) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 23, 0, 0, 0, 0, time.UTC))
}

func (m FixOrphanedPostTextVersions) Name() string {
	return "FixOrphanedPostTextVersions"
}

func (m FixOrphanedPostTextVersions) Description() string {
	return "Set the post_id on posttextversions that lost track of their parents"
}

func (m FixOrphanedPostTextVersions) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		UPDATE handmade_posttextversion AS tver
		SET
			post_id = (
				SELECT id
				FROM handmade_post AS post
				WHERE
					post.current_id = tver.id
			)
		WHERE
			tver.post_id IS NULL
	`)
	if err != nil {
		return oops.New(err, "failed to fix up half-orphaned posttextversions")
	}

	return nil
}

func (m FixOrphanedPostTextVersions) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

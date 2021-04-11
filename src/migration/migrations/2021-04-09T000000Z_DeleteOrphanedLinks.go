package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(DeleteOrphanedLinks{})
}

type DeleteOrphanedLinks struct{}

func (m DeleteOrphanedLinks) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 9, 0, 0, 0, 0, time.UTC))
}

func (m DeleteOrphanedLinks) Name() string {
	return "DeleteOrphanedLinks"
}

func (m DeleteOrphanedLinks) Description() string {
	return "Delete links with no member or project"
}

func (m DeleteOrphanedLinks) Up(tx pgx.Tx) error {
	// Delete orphaned links (no member or project)
	_, err := tx.Exec(context.Background(), `
		DELETE FROM handmade_links
		WHERE
			id IN (
				SELECT links.id
				FROM
					handmade_links AS links
					LEFT JOIN handmade_memberextended_links AS mlinks ON mlinks.links_id = links.id
					LEFT JOIN handmade_project_links AS plinks ON plinks.links_id = links.id
				WHERE
					mlinks.id IS NULL
					AND plinks.id IS NULL
			)
	`)
	if err != nil {
		return oops.New(err, "failed to delete orphaned links")
	}

	return nil
}

func (m DeleteOrphanedLinks) Down(tx pgx.Tx) error {
	panic("Implement me")
}

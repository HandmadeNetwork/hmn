package migrations

import (
	"context"
	"fmt"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(DeleteOrphanedData{})
}

type DeleteOrphanedData struct{}

func (m DeleteOrphanedData) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 9, 0, 0, 0, 0, time.UTC))
}

func (m DeleteOrphanedData) Name() string {
	return "DeleteOrphanedData"
}

func (m DeleteOrphanedData) Description() string {
	return "Delete data that doesn't have other important associated records"
}

func (m DeleteOrphanedData) Up(ctx context.Context, tx pgx.Tx) error {
	// Delete orphaned users (no member)
	res, err := tx.Exec(ctx, `
		DELETE FROM auth_user
		WHERE
			id IN (
				SELECT auth_user.id
				FROM
					auth_user
					LEFT JOIN handmade_member ON auth_user.id = handmade_member.user_id
				WHERE
					handmade_member.user_id IS NULL
			)
	`)
	if err != nil {
		return oops.New(err, "failed to delete users without members")
	}
	fmt.Printf("Deleted %v users\n", res.RowsAffected())

	orphanedMemberExtendedIdsQuery := `
		SELECT mext.id
		FROM
			handmade_memberextended AS mext
			LEFT JOIN handmade_member AS member ON mext.id = member.extended_id
		WHERE
			member.user_id IS NULL
	`

	// Delete memberextended<->links joins for memberextendeds that are about to die
	// (my kingdom for ON DELETE CASCADE, I mean come on)
	res, err = tx.Exec(ctx, `
		DELETE FROM handmade_memberextended_links
		WHERE
			memberextended_id IN (
				`+orphanedMemberExtendedIdsQuery+`
			)
	`)
	if err != nil {
		return oops.New(err, "failed to delete memberextendeds without members")
	}
	fmt.Printf("Deleted %v memberextended<->links joins\n", res.RowsAffected())

	// Delete orphaned memberextendeds (no member)
	res, err = tx.Exec(ctx, `
		DELETE FROM handmade_memberextended
		WHERE
			id IN (
				`+orphanedMemberExtendedIdsQuery+`
			)
	`)
	if err != nil {
		return oops.New(err, "failed to delete memberextendeds without members")
	}
	fmt.Printf("Deleted %v memberextendeds\n", res.RowsAffected())

	// Delete orphaned links (no member or project)
	res, err = tx.Exec(ctx, `
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
	fmt.Printf("Deleted %v links\n", res.RowsAffected())

	return nil
}

func (m DeleteOrphanedData) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

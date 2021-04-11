package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

/*
Phase 1. Migrate the schemas for all the tables that will stick around through this
whole process.
*/

func init() {
	registerMigration(RemoveMemberAndExtended{})
}

type RemoveMemberAndExtended struct{}

func (m RemoveMemberAndExtended) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 11, 19, 57, 47, 0, time.UTC))
}

func (m RemoveMemberAndExtended) Name() string {
	return "RemoveMemberAndExtended"
}

func (m RemoveMemberAndExtended) Description() string {
	return "Remove the member and member extended records, collapsing their data into users"
}

func (m RemoveMemberAndExtended) Up(tx pgx.Tx) error {
	// Creates a column that will eventually be a foreign key to auth_user.
	createUserColumn := func(ctx context.Context, tx pgx.Tx, table string, before string, notNull bool) {
		nullConstraint := ""
		if notNull {
			nullConstraint = "NOT NULL"
		}

		_, err := tx.Exec(ctx, `
			ALTER TABLE `+table+`
				ADD plz_rename INT `+nullConstraint+` DEFAULT 99999;
			UPDATE `+table+` SET plz_rename = `+before+`;
		`)
		if err != nil {
			panic(oops.New(err, "failed to update table %s to point at users instead of members", table))
		}
	}

	/*
		Models referencing handmade_member:
		- CommunicationChoice
		- CommunicationSubCategory
		- CommunicationSubThread
		- Discord
		- CategoryLastReadInfo
		- ThreadLastReadInfo
		- PostTextVersion
		- Post
		- handmade_member_projects
	*/
	createUserColumn(context.Background(), tx, "handmade_communicationchoice", "member_id", true)
	createUserColumn(context.Background(), tx, "handmade_communicationsubcategory", "member_id", true)
	createUserColumn(context.Background(), tx, "handmade_communicationsubthread", "member_id", true)
	createUserColumn(context.Background(), tx, "handmade_discord", "member_id", true)
	createUserColumn(context.Background(), tx, "handmade_categorylastreadinfo", "member_id", true)
	createUserColumn(context.Background(), tx, "handmade_threadlastreadinfo", "member_id", true)
	createUserColumn(context.Background(), tx, "handmade_posttextversion", "editor_id", false)
	createUserColumn(context.Background(), tx, "handmade_post", "author_id", false)

	return nil
}

func (m RemoveMemberAndExtended) Down(tx pgx.Tx) error {
	panic("Implement me")
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

/*
Phase 2. Clean up the schema, adding constraints and dropping tables. Do not do any row updates
or deletes, because that causes trigger events, which make me sad.
*/

func init() {
	registerMigration(RemoveMemberAndExtended2{})
}

type RemoveMemberAndExtended2 struct{}

func (m RemoveMemberAndExtended2) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 11, 21, 9, 58, 0, time.UTC))
}

func (m RemoveMemberAndExtended2) Name() string {
	return "RemoveMemberAndExtended2"
}

func (m RemoveMemberAndExtended2) Description() string {
	return "Phase 2 of the above"
}

func (m RemoveMemberAndExtended2) Up(tx pgx.Tx) error {
	dropOldColumn := func(ctx context.Context, tx pgx.Tx, table string, before, after string, onDelete string) {
		_, err := tx.Exec(ctx, `
			ALTER TABLE `+table+`
				DROP `+before+`;
			ALTER TABLE `+table+`
				RENAME plz_rename TO `+after+`;
			ALTER TABLE `+table+`
				ADD FOREIGN KEY (`+after+`) REFERENCES auth_user ON DELETE `+onDelete+`,
				ALTER `+after+` DROP DEFAULT;
		`)
		if err != nil {
			panic(oops.New(err, "failed to update table %s to point at users instead of members", table))
		}
	}

	dropOldColumn(context.Background(), tx, "handmade_communicationchoice", "member_id", "user_id", "CASCADE")
	dropOldColumn(context.Background(), tx, "handmade_communicationsubcategory", "member_id", "user_id", "CASCADE")
	dropOldColumn(context.Background(), tx, "handmade_communicationsubthread", "member_id", "user_id", "CASCADE")
	dropOldColumn(context.Background(), tx, "handmade_discord", "member_id", "hmn_user_id", "CASCADE")
	dropOldColumn(context.Background(), tx, "handmade_categorylastreadinfo", "member_id", "user_id", "CASCADE")
	dropOldColumn(context.Background(), tx, "handmade_threadlastreadinfo", "member_id", "user_id", "CASCADE")
	dropOldColumn(context.Background(), tx, "handmade_posttextversion", "editor_id", "editor_id", "SET NULL")
	dropOldColumn(context.Background(), tx, "handmade_post", "author_id", "author_id", "SET NULL")

	return nil
}

func (m RemoveMemberAndExtended2) Down(tx pgx.Tx) error {
	panic("Implement me")
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
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

func (m RemoveMemberAndExtended2) Up(ctx context.Context, tx pgx.Tx) error {
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

	dropOldColumn(ctx, tx, "handmade_communicationchoice", "member_id", "user_id", "CASCADE")
	dropOldColumn(ctx, tx, "handmade_communicationsubcategory", "member_id", "user_id", "CASCADE")
	dropOldColumn(ctx, tx, "handmade_communicationsubthread", "member_id", "user_id", "CASCADE")
	dropOldColumn(ctx, tx, "handmade_discord", "member_id", "hmn_user_id", "CASCADE")
	dropOldColumn(ctx, tx, "handmade_categorylastreadinfo", "member_id", "user_id", "CASCADE")
	dropOldColumn(ctx, tx, "handmade_threadlastreadinfo", "member_id", "user_id", "CASCADE")
	dropOldColumn(ctx, tx, "handmade_posttextversion", "editor_id", "editor_id", "SET NULL")
	dropOldColumn(ctx, tx, "handmade_post", "author_id", "author_id", "SET NULL")
	dropOldColumn(ctx, tx, "handmade_member_projects", "member_id", "user_id", "SET NULL")

	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_member_projects
			RENAME TO handmade_user_projects;
	`)
	if err != nil {
		return oops.New(err, "failed to rename member projects table")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_links
			ADD FOREIGN KEY (user_id) REFERENCES auth_user ON DELETE CASCADE,
			ALTER user_id DROP DEFAULT,
			ADD FOREIGN KEY (project_id) REFERENCES handmade_project ON DELETE CASCADE,
			ALTER project_id DROP DEFAULT,
			ADD CONSTRAINT exactly_one_foreign_key CHECK (
				(
					CASE WHEN user_id IS NOT NULL THEN 1 ELSE 0 END
					+ CASE WHEN project_id IS NOT NULL THEN 1 ELSE 0 END
				) = 1
			);

		DROP TABLE handmade_memberextended_links;
		DROP TABLE handmade_project_links;
	`)
	if err != nil {
		return oops.New(err, "failed to add constraints to new handmade_links columns")
	}

	// And now, the moment you've all been waiting for:
	_, err = tx.Exec(ctx, `
		DROP TABLE handmade_member;
		DROP TABLE handmade_memberextended;
	`)
	if err != nil {
		return oops.New(err, "failed to delete those damn tables")
	}

	// And finally, a little cleanup.
	_, err = tx.Exec(ctx, `
		ALTER INDEX handmade_member_projects_b098ad43 RENAME TO user_projects_btree;
		ALTER SEQUENCE handmade_member_projects_id_seq RENAME TO user_projects_id_seq;
		ALTER INDEX handmade_member_projects_pkey RENAME TO user_projects_pkey;
	`)
	if err != nil {
		return oops.New(err, "failed to rename some indexes and stuff")
	}

	return nil
}

func (m RemoveMemberAndExtended2) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

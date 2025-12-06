package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
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

func (m RemoveMemberAndExtended) Up(ctx context.Context, tx pgx.Tx) error {
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

	// Migrate a lot of simple foreign keys
	createUserColumn(ctx, tx, "handmade_communicationchoice", "member_id", true)
	createUserColumn(ctx, tx, "handmade_communicationsubcategory", "member_id", true)
	createUserColumn(ctx, tx, "handmade_communicationsubthread", "member_id", true)
	createUserColumn(ctx, tx, "handmade_discord", "member_id", true)
	createUserColumn(ctx, tx, "handmade_categorylastreadinfo", "member_id", true)
	createUserColumn(ctx, tx, "handmade_threadlastreadinfo", "member_id", true)
	createUserColumn(ctx, tx, "handmade_posttextversion", "editor_id", false)
	createUserColumn(ctx, tx, "handmade_post", "author_id", false)
	createUserColumn(ctx, tx, "handmade_member_projects", "member_id", true)

	// Directly associate links with members
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_links
			ADD COLUMN user_id INTEGER DEFAULT 99999,
			ADD COLUMN project_id INTEGER DEFAULT 99999;
		
		UPDATE handmade_links
		SET (user_id) = (
			SELECT mem.user_id
			FROM
				handmade_memberextended_links AS mlinks
				JOIN handmade_memberextended AS memext ON memext.id = mlinks.memberextended_id
				JOIN handmade_member AS mem ON mem.extended_id = memext.id
			WHERE
				mlinks.links_id = handmade_links.id
		);

		UPDATE handmade_links
		SET (project_id) = (
			SELECT proj.id
			FROM
				handmade_project_links AS plinks
				JOIN handmade_project AS proj ON proj.id = plinks.project_id
			WHERE
				plinks.links_id = handmade_links.id
		);
	`)
	if err != nil {
		return oops.New(err, "failed to associate links with members and projects")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE auth_user
			-- From handmade_member --
			ADD blurb VARCHAR(140) NOT NULL DEFAULT '',
			ADD name VARCHAR(255) NOT NULL DEFAULT '',
			ADD signature TEXT NOT NULL DEFAULT '',
			ADD avatar VARCHAR(100),
			ADD location VARCHAR(100) NOT NULL DEFAULT '',
			-- ordering is dropped
			-- posts is dropped
			-- profileviews is dropped
			-- thanked is dropped
			ADD timezone VARCHAR(255),
			ADD color_1 VARCHAR(6),
			ADD color_2 VARCHAR(6),
			ADD darktheme BOOLEAN NOT NULL DEFAULT FALSE,
			-- extended_id is dropped
			-- project_count_all is dropped
			-- project_count_public is dropped
			-- matrix_username is dropped
			-- set_matrix_display_name is dropped
			ADD edit_library BOOLEAN NOT NULL DEFAULT FALSE,
			ADD discord_delete_snippet_on_message_delete BOOLEAN NOT NULL DEFAULT TRUE,
			ADD discord_save_showcase BOOLEAN NOT NULL DEFAULT TRUE,

			-- From handmade_memberextended --
			ADD bio TEXT NOT NULL DEFAULT '',
			ADD showemail BOOLEAN NOT NULL DEFAULT FALSE;
			-- sendemail is dropped
			-- joomlaid is dropped
			-- lastResetTime is dropped
			-- resetCount is dropped
			-- requireReset is dropped
			-- birthdate is dropped
		
		UPDATE auth_user
		SET (
			blurb,
			name,
			signature,
			avatar,
			location,
			timezone,
			color_1,
			color_2,
			darktheme,
			edit_library,
			discord_delete_snippet_on_message_delete,
			discord_save_showcase,
			bio,
			showemail
		) = (
			SELECT
				COALESCE(blurb, ''),
				COALESCE(name, ''),
				COALESCE(signature, ''),
				avatar,
				COALESCE(location, ''),
				timezone,
				color_1,
				color_2,
				darktheme,
				edit_library,
				discord_delete_snippet_on_message_delete,
				discord_save_showcase,
				COALESCE(bio, ''),
				showemail
			FROM
				handmade_member AS mem
				JOIN handmade_memberextended AS mext ON mem.extended_id = mext.id
			WHERE
				mem.user_id = auth_user.id
		);
		
		ALTER TABLE auth_user
			ALTER timezone SET NOT NULL,
			ALTER color_1 SET NOT NULL,
			ALTER color_2 SET NOT NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to copy fields to auth_user")
	}

	return nil
}

func (m RemoveMemberAndExtended) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

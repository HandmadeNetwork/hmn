package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	// registerMigration(CopyMemberExtendedData{})
}

type CopyMemberExtendedData struct{}

func (m CopyMemberExtendedData) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 10, 1, 30, 44, 0, time.UTC))
}

func (m CopyMemberExtendedData) Name() string {
	return "CopyMemberExtendedData"
}

func (m CopyMemberExtendedData) Description() string {
	return "Copy MemberExtended data into Member"
}

func (m CopyMemberExtendedData) Up(tx pgx.Tx) error {
	// Add columns to member table
	_, err := tx.Exec(context.Background(), `
		ALTER TABLE handmade_member
			ADD COLUMN bio TEXT NOT NULL DEFAULT '',
			ADD COLUMN showemail BOOLEAN NOT NULL DEFAULT FALSE
	`)
	if err != nil {
		return oops.New(err, "failed to add columns to member table")
	}

	// Copy data to members from memberextended
	_, err = tx.Exec(context.Background(), `
		UPDATE handmade_member
		SET (bio, showemail) = (
			SELECT COALESCE(bio, ''), showemail
			FROM handmade_memberextended
			WHERE handmade_memberextended.id = handmade_member.extended_id
		);
	`)
	if err != nil {
		return oops.New(err, "failed to copy data from the memberextended table")
	}

	// Directly associate links with members
	_, err = tx.Exec(context.Background(), `
		ALTER TABLE handmade_links
			ADD COLUMN member_id INTEGER REFERENCES handmade_member,
			ADD COLUMN project_id INTEGER REFERENCES handmade_project,
			ADD CONSTRAINT exactly_one_foreign_key CHECK (
				(
					CASE WHEN member_id IS NULL THEN 0 ELSE 1 END
					+ CASE WHEN project_id IS NULL THEN 0 ELSE 1 END
				) = 1
			);

		UPDATE handmade_links
		SET (member_id) = (
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
		return oops.New(err, "failed to associate links with members")
	}

	return nil
}

func (m CopyMemberExtendedData) Down(tx pgx.Tx) error {
	// _, err := tx.Exec(context.Background(), `
	// 	ALTER TABLE handmade_member
	// 		DROP COLUMN bio,
	// 		DROP COLUMN showemail
	// `)
	// if err != nil {
	// 	return oops.New(err, "failed to drop columns from member table")
	// }
	//
	// return nil

	panic("you do not want to do this")
}

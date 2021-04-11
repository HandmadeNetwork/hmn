package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v4"
)

func init() {
	// TODO: Delete this migration
	// registerMigration(DropMemberExtendedTable{})
}

type DropMemberExtendedTable struct{}

func (m DropMemberExtendedTable) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 10, 1, 53, 39, 0, time.UTC))
}

func (m DropMemberExtendedTable) Name() string {
	return "DropMemberExtendedTable"
}

func (m DropMemberExtendedTable) Description() string {
	return "Remove the MemberExtended record outright"
}

func (m DropMemberExtendedTable) Up(tx pgx.Tx) error {
	_, err := tx.Exec(context.Background(), `
		ALTER TABLE handmade_member
			DROP COLUMN extended_id;

		DROP TABLE handmade_memberextended_links;
		DROP TABLE handmade_memberextended;
	`)
	return err
}

func (m DropMemberExtendedTable) Down(tx pgx.Tx) error {
	panic("you do not want to do this")
}

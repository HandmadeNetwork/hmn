package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(RemoveParserField{})
}

type RemoveParserField struct{}

func (m RemoveParserField) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 3, 20, 2, 33, 0, time.UTC))
}

func (m RemoveParserField) Name() string {
	return "RemoveParserField"
}

func (m RemoveParserField) Description() string {
	return "Remove the parser field on post versions, since we now have a universal parser"
}

func (m RemoveParserField) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_postversion
			DROP parser;
	`)
	if err != nil {
		return oops.New(err, "failed to delete parser field")
	}

	return nil
}

func (m RemoveParserField) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(ReworkPostFields{})
}

type ReworkPostFields struct{}

func (m ReworkPostFields) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 4, 20, 46, 42, 0, time.UTC))
}

func (m ReworkPostFields) Name() string {
	return "ReworkPostFields"
}

func (m ReworkPostFields) Description() string {
	return "Clean up post and postversion fields"
}

func (m ReworkPostFields) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_post
			DROP depth,
			DROP slug,
			DROP author_name,
			DROP ip,
			DROP sticky,
			DROP hits,
			DROP featured,
			DROP featurevotes;
	`)
	if err != nil {
		return oops.New(err, "failed to drop unnecessary post fields")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_postversion
			RENAME edit_ip TO ip;
		ALTER TABLE handmade_postversion
			RENAME edit_date TO date;
	`)
	if err != nil {
		return oops.New(err, "failed to rename postversion fields")
	}

	_, err = tx.Exec(ctx, `
		DROP TABLE handmade_kunenapost;
	`)
	if err != nil {
		return oops.New(err, "failed to drop ancient weirdo table")
	}

	return nil
}

func (m ReworkPostFields) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

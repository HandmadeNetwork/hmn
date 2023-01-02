package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(ImageFileNotNull{})
}

type ImageFileNotNull struct{}

func (m ImageFileNotNull) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 9, 8, 0, 43, 57, 0, time.UTC))
}

func (m ImageFileNotNull) Name() string {
	return "ImageFileNotNull"
}

func (m ImageFileNotNull) Description() string {
	return "Don't allow the filename of an image file to be null"
}

func (m ImageFileNotNull) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_imagefile
			ALTER file SET NOT NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to make imagefile filename not nullable")
	}

	return nil
}

func (m ImageFileNotNull) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_imagefile
			ALTER file SET NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to make imagefile filename nullable again")
	}

	return nil
}

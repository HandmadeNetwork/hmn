package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(RenameModeratedFields{})
}

type RenameModeratedFields struct{}

func (m RenameModeratedFields) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 29, 4, 35, 48, 0, time.UTC))
}

func (m RenameModeratedFields) Name() string {
	return "RenameModeratedFields"
}

func (m RenameModeratedFields) Description() string {
	return "Rename 'moderated' to 'deleted'"
}

func (m RenameModeratedFields) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_thread
			RENAME moderated TO deleted;
		ALTER TABLE handmade_post
			RENAME moderated TO deleted;
	`)
	if err != nil {
		return oops.New(err, "failed to rename columns")
	}

	_, err = tx.Exec(ctx,
		`
		ALTER TABLE handmade_thread
			ALTER deleted TYPE bool USING CASE WHEN deleted = 0 THEN FALSE ELSE TRUE END;
		ALTER TABLE handmade_thread ALTER COLUMN deleted SET DEFAULT FALSE;

		ALTER TABLE handmade_post ALTER COLUMN deleted SET DEFAULT FALSE;
		`,
	)
	if err != nil {
		return oops.New(err, "failed to convert ints to bools")
	}

	return nil
}

func (m RenameModeratedFields) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

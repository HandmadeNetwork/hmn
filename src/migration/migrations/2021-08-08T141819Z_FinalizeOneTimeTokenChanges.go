package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(FinalizeOneTimeTokenChanges{})
}

type FinalizeOneTimeTokenChanges struct{}

func (m FinalizeOneTimeTokenChanges) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 8, 14, 18, 19, 0, time.UTC))
}

func (m FinalizeOneTimeTokenChanges) Name() string {
	return "FinalizeOneTimeTokenChanges"
}

func (m FinalizeOneTimeTokenChanges) Description() string {
	return "Create an index and set not-null"
}

func (m FinalizeOneTimeTokenChanges) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_onetimetoken
			ALTER COLUMN owner_id SET NOT NULL;

		CREATE INDEX handmade_onetimetoken_ownerid_type ON handmade_onetimetoken(owner_id, token_type);
	`)
	if err != nil {
		return oops.New(err, "Create index on onetimetoken")
	}

	_, err = tx.Exec(ctx, `
		DROP TABLE handmade_userpending;
	`)
	if err != nil {
		return oops.New(err, "failed to drop handmade_userpending")
	}

	_, err = tx.Exec(ctx, `
		DROP TABLE handmade_passwordresetrequest;
	`)
	if err != nil {
		return oops.New(err, "failed to drop handmade_passwordresetrequest")
	}

	return nil
}

func (m FinalizeOneTimeTokenChanges) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

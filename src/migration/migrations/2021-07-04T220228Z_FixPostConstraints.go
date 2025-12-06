package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(FixPostConstraints{})
}

type FixPostConstraints struct{}

func (m FixPostConstraints) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 4, 22, 2, 28, 0, time.UTC))
}

func (m FixPostConstraints) Name() string {
	return "FixPostConstraints"
}

func (m FixPostConstraints) Description() string {
	return "Update post-related constraints to make insertion sane"
}

func (m FixPostConstraints) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_thread
			ALTER locked SET DEFAULT FALSE,
			ALTER first_id SET NOT NULL,
			ALTER last_id SET NOT NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to update thread constraints")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_post
			ALTER deleted SET DEFAULT FALSE,
			ALTER readonly SET DEFAULT FALSE,
			ALTER CONSTRAINT handmade_post_current_id_fkey DEFERRABLE INITIALLY DEFERRED;
	`)
	if err != nil {
		return oops.New(err, "failed to update project constraints")
	}

	_, err = tx.Exec(ctx, `
		CREATE SEQUENCE handmade_postversion_id_seq
			START WITH 40000 -- this is well out of the way of existing IDs
			OWNED BY handmade_postversion.id;

		ALTER TABLE handmade_postversion
			ALTER id SET DEFAULT nextval('handmade_postversion_id_seq'),
			ALTER CONSTRAINT handmade_postversion_post_id_fkey DEFERRABLE INITIALLY DEFERRED;
	`)
	if err != nil {
		return oops.New(err, "failed to update postversion constraints")
	}

	return nil
}

func (m FixPostConstraints) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

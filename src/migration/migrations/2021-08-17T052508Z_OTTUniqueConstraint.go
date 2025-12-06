package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(OTTUniqueConstraint{})
}

type OTTUniqueConstraint struct{}

func (m OTTUniqueConstraint) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 17, 5, 25, 8, 0, time.UTC))
}

func (m OTTUniqueConstraint) Name() string {
	return "OTTUniqueConstraint"
}

func (m OTTUniqueConstraint) Description() string {
	return "Add userid+type unique constraint to onetimetoken"
}

func (m OTTUniqueConstraint) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP INDEX handmade_onetimetoken_ownerid_type;
		ALTER TABLE handmade_onetimetoken
			ADD CONSTRAINT handmade_onetimetoken_ownerid_type UNIQUE(owner_id, token_type);
		`,
	)
	if err != nil {
		return oops.New(err, "failed to replace index with unique constraint")
	}
	return nil
}

func (m OTTUniqueConstraint) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

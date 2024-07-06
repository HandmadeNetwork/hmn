package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddFeaturedToUser{})
}

type AddFeaturedToUser struct{}

func (m AddFeaturedToUser) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 7, 6, 15, 34, 32, 0, time.UTC))
}

func (m AddFeaturedToUser) Name() string {
	return "AddFeaturedToUser"
}

func (m AddFeaturedToUser) Description() string {
	return "Add featured flag to users"
}

func (m AddFeaturedToUser) Up(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE hmn_user
			ADD COLUMN featured BOOLEAN NOT NULL DEFAULT false;
		CREATE INDEX hmn_user_featured ON hmn_user(featured);
		`,
	))
	return nil
}

func (m AddFeaturedToUser) Down(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		DROP INDEX hmn_user_featured;
		ALTER TABLE hmn_user
			DROP COLUMN featured;
		`,
	))
	return nil
}

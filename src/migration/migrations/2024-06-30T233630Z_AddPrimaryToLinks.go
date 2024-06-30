package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddPrimaryToLinks{})
}

type AddPrimaryToLinks struct{}

func (m AddPrimaryToLinks) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 6, 30, 23, 36, 30, 0, time.UTC))
}

func (m AddPrimaryToLinks) Name() string {
	return "AddPrimaryToLinks"
}

func (m AddPrimaryToLinks) Description() string {
	return "Adds 'primary_link' field to links"
}

func (m AddPrimaryToLinks) Up(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE link
			ADD COLUMN primary_link BOOLEAN NOT NULL DEFAULT false;
		`,
	))
	return nil
}

func (m AddPrimaryToLinks) Down(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE link
			DROP COLUMN primary_link;
		`,
	))
	return nil
}

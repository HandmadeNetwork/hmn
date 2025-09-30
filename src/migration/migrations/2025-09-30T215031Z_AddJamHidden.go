package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddJamHidden{})
}

type AddJamHidden struct{}

func (m AddJamHidden) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2025, 9, 30, 21, 50, 31, 0, time.UTC))
}

func (m AddJamHidden) Name() string {
	return "AddJamHidden"
}

func (m AddJamHidden) Description() string {
	return "Add flag to hide project in jam pages"
}

func (m AddJamHidden) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project
			ADD COLUMN jam_hidden BOOLEAN NOT NULL DEFAULT FALSE;
		`,
	)
	return err
}

func (m AddJamHidden) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project
			DROP COLUMN jam_hidden;
		`,
	)
	return err
}

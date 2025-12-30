package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddSortScore{})
}

type AddSortScore struct{}

func (m AddSortScore) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2025, 12, 30, 17, 59, 1, 0, time.UTC))
}

func (m AddSortScore) Name() string {
	return "AddSortScore"
}

func (m AddSortScore) Description() string {
	return "Add sort score to project for manual sorting on the project page"
}

func (m AddSortScore) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project
			ADD COLUMN sort_score INTEGER NOT NULL DEFAULT 0;
		`,
	)
	return err
}

func (m AddSortScore) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project
			DROP COLUMN sort_score;
		`,
	)
	return err
}

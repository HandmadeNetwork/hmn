package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(RemoveDarkTheme{})
}

type RemoveDarkTheme struct{}

func (m RemoveDarkTheme) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 6, 18, 2, 25, 36, 0, time.UTC))
}

func (m RemoveDarkTheme) Name() string {
	return "RemoveDarkTheme"
}

func (m RemoveDarkTheme) Description() string {
	return "Remove the darktheme field from users"
}

func (m RemoveDarkTheme) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE hmn_user
			DROP COLUMN darktheme;
		`,
	)
	return err
}

func (m RemoveDarkTheme) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE hmn_user
			ADD COLUMN darktheme BOOLEAN NOT NULL DEFAULT FALSE;
		`,
	)
	return err
}

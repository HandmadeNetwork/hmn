package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(RemoveProjectLogoUrls{})
}

type RemoveProjectLogoUrls struct{}

func (m RemoveProjectLogoUrls) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 2, 13, 20, 1, 55, 0, time.UTC))
}

func (m RemoveProjectLogoUrls) Name() string {
	return "RemoveProjectLogoUrls"
}

func (m RemoveProjectLogoUrls) Description() string {
	return "Remove project logo url fields as we're now using assets"
}

func (m RemoveProjectLogoUrls) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_project
			DROP COLUMN logolight,
			DROP COLUMN logodark;
		`,
	)
	return err
}

func (m RemoveProjectLogoUrls) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_project
			ADD COLUMN logolight character varying(100),
			ADD COLUMN logodark character varying(100);
		`,
	)
	return err
}

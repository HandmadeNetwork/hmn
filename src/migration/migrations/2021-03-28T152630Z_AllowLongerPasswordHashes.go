package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AllowLongerPasswordHashes{})
}

type AllowLongerPasswordHashes struct{}

func (m AllowLongerPasswordHashes) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 3, 28, 15, 26, 30, 0, time.UTC))
}

func (m AllowLongerPasswordHashes) Name() string {
	return "AllowLongerPasswordHashes"
}

func (m AllowLongerPasswordHashes) Description() string {
	return "Increase the storage size limit on hashed passwords"
}

func (m AllowLongerPasswordHashes) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE auth_user
			ALTER COLUMN password TYPE VARCHAR(256)
	`)
	return err
}

func (m AllowLongerPasswordHashes) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE auth_user
			ALTER COLUMN password TYPE VARCHAR(128)
	`)
	return err
}

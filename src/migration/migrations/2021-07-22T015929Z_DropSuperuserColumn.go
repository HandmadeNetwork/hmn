package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(DropSuperuserColumn{})
}

type DropSuperuserColumn struct{}

func (m DropSuperuserColumn) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 22, 1, 59, 29, 0, time.UTC))
}

func (m DropSuperuserColumn) Name() string {
	return "DropSuperuserColumn"
}

func (m DropSuperuserColumn) Description() string {
	return "Drop the is_superuser column on users, in favor of is_staff"
}

func (m DropSuperuserColumn) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE auth_user
			DROP is_superuser;
	`)
	if err != nil {
		return oops.New(err, "failed to drop superuser column")
	}

	return nil
}

func (m DropSuperuserColumn) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

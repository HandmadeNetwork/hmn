package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(UsernameUniqueIndex{})
}

type UsernameUniqueIndex struct{}

func (m UsernameUniqueIndex) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 27, 3, 43, 27, 0, time.UTC))
}

func (m UsernameUniqueIndex) Name() string {
	return "UsernameUniqueIndex"
}

func (m UsernameUniqueIndex) Description() string {
	return "Prevent the creation of similar usernames with different character cases"
}

func (m UsernameUniqueIndex) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `CREATE UNIQUE INDEX auth_user_unique_username_case_insensitive ON auth_user (LOWER(username))`)
	if err != nil {
		return oops.New(err, "failed to add unique index")
	}
	return nil
}

func (m UsernameUniqueIndex) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

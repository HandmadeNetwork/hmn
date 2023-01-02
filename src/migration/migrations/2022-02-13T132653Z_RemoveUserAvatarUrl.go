package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(RemoveUserAvatarUrl{})
}

type RemoveUserAvatarUrl struct{}

func (m RemoveUserAvatarUrl) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 2, 13, 13, 26, 53, 0, time.UTC))
}

func (m RemoveUserAvatarUrl) Name() string {
	return "RemoveUserAvatarUrl"
}

func (m RemoveUserAvatarUrl) Description() string {
	return "Remove avatar url field from users as we're using assets now"
}

func (m RemoveUserAvatarUrl) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE auth_user
			DROP COLUMN avatar;
		`,
	)
	return err
}

func (m RemoveUserAvatarUrl) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE auth_user
			ADD COLUMN avatar character varying(100);
		`,
	)
	return err
}

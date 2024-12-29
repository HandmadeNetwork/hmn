package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddBouncedAtToEmailBlacklist{})
}

type AddBouncedAtToEmailBlacklist struct{}

func (m AddBouncedAtToEmailBlacklist) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 12, 29, 11, 0, 44, 0, time.UTC))
}

func (m AddBouncedAtToEmailBlacklist) Name() string {
	return "AddBouncedAtToEmailBlacklist"
}

func (m AddBouncedAtToEmailBlacklist) Description() string {
	return "Add bounced_at to email blacklist"
}

func (m AddBouncedAtToEmailBlacklist) Up(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE email_blacklist
			ADD COLUMN bounced_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW();
		`,
	))
	return nil
}

func (m AddBouncedAtToEmailBlacklist) Down(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE email_blacklist
			DROP bounced_at;
		`,
	))
	return nil
}

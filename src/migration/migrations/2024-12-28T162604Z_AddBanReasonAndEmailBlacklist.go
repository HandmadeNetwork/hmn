package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddBanReasonAndEmailBlacklist{})
}

type AddBanReasonAndEmailBlacklist struct{}

func (m AddBanReasonAndEmailBlacklist) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 12, 28, 16, 26, 4, 0, time.UTC))
}

func (m AddBanReasonAndEmailBlacklist) Name() string {
	return "AddBanReasonAndEmailBlacklist"
}

func (m AddBanReasonAndEmailBlacklist) Description() string {
	return "Adds ban reason to hmn_user and adds a blacklist table"
}

func (m AddBanReasonAndEmailBlacklist) Up(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE hmn_user
			ADD COLUMN ban_reason text NOT NULL DEFAULT '';

		CREATE TABLE email_blacklist (
			email text NOT NULL PRIMARY KEY,
			blacklisted_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			reason text NOT NULL DEFAULT '',
			details text NOT NULL DEFAULT ''
		);

		CREATE INDEX blacklisted_at_index ON email_blacklist(blacklisted_at);
		`,
	))
	return nil
}

func (m AddBanReasonAndEmailBlacklist) Down(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE hmn_user
			DROP ban_reason;

		DROP TABLE email_blacklist;
		`,
	))
	return nil
}

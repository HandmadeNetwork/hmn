package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(TwitchIgnoreList{})
}

type TwitchIgnoreList struct{}

func (m TwitchIgnoreList) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2025, 7, 14, 15, 15, 57, 0, time.UTC))
}

func (m TwitchIgnoreList) Name() string {
	return "TwitchIgnoreList"
}

func (m TwitchIgnoreList) Description() string {
	return "Adds a twitch ignore list table"
}

func (m TwitchIgnoreList) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE twitch_ignore_list (
			twitch_login TEXT NOT NULL PRIMARY KEY,
			banned BOOLEAN DEFAULT TRUE
		);
		`,
	)

	return err
}

func (m TwitchIgnoreList) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP TABLE twitch_ignore_list;
		`,
	)

	return err
}

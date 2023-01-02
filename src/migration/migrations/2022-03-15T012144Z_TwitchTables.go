package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(TwitchTables{})
}

type TwitchTables struct{}

func (m TwitchTables) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 3, 15, 1, 21, 44, 0, time.UTC))
}

func (m TwitchTables) Name() string {
	return "TwitchTables"
}

func (m TwitchTables) Description() string {
	return "Create tables for live twitch streams and twitch ID cache"
}

func (m TwitchTables) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE twitch_streams (
			twitch_id VARCHAR(255) NOT NULL,
			twitch_login VARCHAR(255) NOT NULL,
			title VARCHAR(255) NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE
		);
		`,
	)

	if err != nil {
		return oops.New(err, "failed to create twitch tables")
	}
	return err
}

func (m TwitchTables) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP TABLE twitch_ids;
		DROP TABLE twitch_streams;
		`,
	)

	if err != nil {
		return oops.New(err, "failed to create twitch tables")
	}
	return err
}

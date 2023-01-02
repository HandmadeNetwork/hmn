package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddTwitchLog{})
}

type AddTwitchLog struct{}

func (m AddTwitchLog) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 8, 28, 20, 39, 35, 0, time.UTC))
}

func (m AddTwitchLog) Name() string {
	return "AddTwitchLog"
}

func (m AddTwitchLog) Description() string {
	return "Add twitch logging"
}

func (m AddTwitchLog) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE twitch_log (
			id SERIAL NOT NULL PRIMARY KEY,
			logged_at TIMESTAMP WITH TIME ZONE NOT NULL,
			twitch_login VARCHAR(256) NOT NULL DEFAULT '',
			type INT NOT NULL DEFAULT 0,
			message TEXT NOT NULL DEFAULT '',
			payload TEXT NOT NULL DEFAULT ''
		);
		`,
	)
	return err
}

func (m AddTwitchLog) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP TABLE twitch_log;
		`,
	)
	return err
}

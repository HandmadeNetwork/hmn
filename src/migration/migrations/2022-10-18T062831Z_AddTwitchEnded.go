package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddTwitchEnded{})
}

type AddTwitchEnded struct{}

func (m AddTwitchEnded) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 10, 18, 6, 28, 31, 0, time.UTC))
}

func (m AddTwitchEnded) Name() string {
	return "AddTwitchEnded"
}

func (m AddTwitchEnded) Description() string {
	return "Add stream_ended to twitch history"
}

func (m AddTwitchEnded) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE twitch_stream_history
		ADD COLUMN stream_ended BOOLEAN NOT NULL DEFAULT FALSE;
		`,
	)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx,
		`
		UPDATE twitch_stream_history
		SET stream_ended = TRUE
		WHERE ended_at > TIMESTAMP '2000-01-01 00:00:00';
		`,
	)
	return err
}

func (m AddTwitchEnded) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE twitch_stream_history
		DROP COLUMN stream_ended BOOLEAN NOT NULL DEFAULT FALSE;
		`,
	)
	return err
}

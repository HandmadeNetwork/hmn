package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddBackfillToDiscordMessage{})
}

type AddBackfillToDiscordMessage struct{}

func (m AddBackfillToDiscordMessage) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 3, 28, 18, 41, 7, 0, time.UTC))
}

func (m AddBackfillToDiscordMessage) Name() string {
	return "AddBackfillToDiscordMessage"
}

func (m AddBackfillToDiscordMessage) Description() string {
	return "Add a backfill flag to discord messages"
}

func (m AddBackfillToDiscordMessage) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE discord_message
		ADD COLUMN backfilled BOOLEAN NOT NULL default FALSE;
		`,
	)
	return err
}

func (m AddBackfillToDiscordMessage) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE discord_message
		DROP COLUMN backfilled;
		`,
	)
	return err
}

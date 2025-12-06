package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddDiscordBotTables{})
}

type AddDiscordBotTables struct{}

func (m AddDiscordBotTables) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 7, 18, 53, 30, 0, time.UTC))
}

func (m AddDiscordBotTables) Name() string {
	return "AddDiscordBotTables"
}

func (m AddDiscordBotTables) Description() string {
	return "Add tables for Discord bot sessions and messages"
}

func (m AddDiscordBotTables) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		CREATE TABLE discord_session (
			pk INT NOT NULL DEFAULT 1337 PRIMARY KEY, -- this should always be set to 1337 to ensure that we only have one row :)
			session_id VARCHAR(255) NOT NULL,
			sequence_number INT NOT NULL,
			
			CONSTRAINT only_one_session CHECK (pk = 1337)
		);
	`)
	if err != nil {
		return oops.New(err, "failed to create discord session table")
	}

	_, err = tx.Exec(ctx, `
		CREATE TABLE discord_outgoingmessages (
			id SERIAL NOT NULL PRIMARY KEY,
			channel_id VARCHAR(64) NOT NULL,
			payload_json TEXT NOT NULL,
			expires_at TIMESTAMP WITH TIME ZONE NOT NULL
		);
	`)
	if err != nil {
		return oops.New(err, "failed to create discord outgoing messages table")
	}

	return nil
}

func (m AddDiscordBotTables) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

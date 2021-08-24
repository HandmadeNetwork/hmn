package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(DiscordDefaults{})
}

type DiscordDefaults struct{}

func (m DiscordDefaults) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 23, 23, 5, 59, 0, time.UTC))
}

func (m DiscordDefaults) Name() string {
	return "DiscordDefaults"
}

func (m DiscordDefaults) Description() string {
	return "Add some default values to Discord fields"
}

func (m DiscordDefaults) Up(ctx context.Context, tx pgx.Tx) error {
	var err error

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_discordmessage
			ALTER snippet_created SET DEFAULT FALSE;
	`)
	if err != nil {
		return oops.New(err, "failed to set message defaults")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_snippet
			ALTER "when" SET DEFAULT NOW(),
			ALTER edited_on_website SET DEFAULT FALSE;
	`)
	if err != nil {
		return oops.New(err, "failed to set snippet defaults")
	}

	return nil
}

func (m DiscordDefaults) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

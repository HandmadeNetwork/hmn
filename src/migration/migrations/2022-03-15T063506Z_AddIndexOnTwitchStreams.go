package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddIndexOnTwitchStreams{})
}

type AddIndexOnTwitchStreams struct{}

func (m AddIndexOnTwitchStreams) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 3, 15, 6, 35, 6, 0, time.UTC))
}

func (m AddIndexOnTwitchStreams) Name() string {
	return "AddIndexOnTwitchStreams"
}

func (m AddIndexOnTwitchStreams) Description() string {
	return "Add unique index on twitch streams"
}

func (m AddIndexOnTwitchStreams) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE UNIQUE INDEX twitch_streams_twitch_id ON twitch_streams (twitch_id);
		`,
	)
	return err
}

func (m AddIndexOnTwitchStreams) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP INDEX twitch_streams_twitch_id;
		`,
	)
	return err
}

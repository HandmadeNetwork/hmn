package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(NewTwitchTracking{})
}

type NewTwitchTracking struct{}

func (m NewTwitchTracking) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 9, 14, 11, 46, 45, 0, time.UTC))
}

func (m NewTwitchTracking) Name() string {
	return "NewTwitchTracking"
}

func (m NewTwitchTracking) Description() string {
	return "New table for twitch tracking"
}

func (m NewTwitchTracking) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE twitch_stream_history (
			stream_id VARCHAR(255) NOT NULL PRIMARY KEY,
			twitch_id VARCHAR(255) NOT NULL,
			twitch_login VARCHAR(255) NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE NOT NULL,
			ended_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch',
			end_approximated BOOLEAN NOT NULL DEFAULT FALSE,
			title VARCHAR(255) NOT NULL DEFAULT '',
			category_id VARCHAR(255) NOT NULL DEFAULT '',
			tags VARCHAR(255) ARRAY NOT NULL DEFAULT '{}',
			discord_message_id VARCHAR(255) NOT NULL DEFAULT '',
			discord_needs_update BOOLEAN NOT NULL DEFAULT FALSE,
			vod_id VARCHAR(255) NOT NULL DEFAULT '',
			vod_url VARCHAR(512) NOT NULL DEFAULT '',
			vod_thumbnail VARCHAR(512) NOT NULL DEFAULT '',
			last_verified_vod TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch',
			vod_gone BOOLEAN NOT NULL DEFAULT FALSE
		);
		CREATE TABLE twitch_latest_status (
			twitch_id VARCHAR(255) NOT NULL PRIMARY KEY,
			twitch_login VARCHAR(255) NOT NULL,
			stream_id VARCHAR(255) NOT NULL DEFAULT '',
			live BOOLEAN NOT NULL DEFAULT FALSE,
			started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch',
			title VARCHAR(255) NOT NULL DEFAULT '',
			category_id VARCHAR(255) NOT NULL DEFAULT '',
			tags VARCHAR(255) ARRAY NOT NULL DEFAULT '{}',
			last_hook_live_update TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch',
			last_hook_channel_update TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch',
			last_rest_update TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch'
		);

		DROP TABLE twitch_stream;
		`,
	)
	return err
}

func (m NewTwitchTracking) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP TABLE twitch_stream_history;
		DROP TABLE twitch_latest_status;

		CREATE TABLE twitch_stream (
			twitch_id VARCHAR(255) NOT NULL,
			twitch_login VARCHAR(255) NOT NULL,
			title VARCHAR(255) NOT NULL,
			started_at TIMESTAMP WITH TIME ZONE
		);
		`,
	)
	return err
}

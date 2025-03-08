package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddPodcastSeason{})
}

type AddPodcastSeason struct{}

func (m AddPodcastSeason) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2025, 3, 8, 14, 55, 33, 0, time.UTC))
}

func (m AddPodcastSeason) Name() string {
	return "AddPodcastSeason"
}

func (m AddPodcastSeason) Description() string {
	return "Make season_number mandatory for podcast episodes"
}

func (m AddPodcastSeason) Up(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		UPDATE podcast_episode SET season_number = 1;
		ALTER TABLE podcast_episode
			ALTER COLUMN season_number SET NOT NULL;
		`,
	))
	return nil
}

func (m AddPodcastSeason) Down(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE podcast_episode
			ALTER COLUMN season_number DROP NOT NULL;
		UPDATE podcast_episode SET season_number = NULL;
		`,
	))
	return nil
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(RenamePodcastColumns{})
}

type RenamePodcastColumns struct{}

func (m RenamePodcastColumns) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 11, 6, 8, 38, 0, time.UTC))
}

func (m RenamePodcastColumns) Name() string {
	return "RenamePodcastColumns"
}

func (m RenamePodcastColumns) Description() string {
	return "Rename columns to lowercase"
}

func (m RenamePodcastColumns) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_podcastepisode
			RENAME COLUMN "enclosureFile" TO "audio_filename";

		ALTER TABLE handmade_podcastepisode
			RENAME COLUMN "pubDate" TO "pub_date";

		ALTER TABLE handmade_podcastepisode
			RENAME COLUMN "episodeNumber" TO "episode_number";

		ALTER TABLE handmade_podcastepisode
			RENAME COLUMN "seasonNumber" TO "season_number";
	`)
	if err != nil {
		return oops.New(err, "failed to rename podcast episode columns")
	}

	return nil
}

func (m RenamePodcastColumns) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

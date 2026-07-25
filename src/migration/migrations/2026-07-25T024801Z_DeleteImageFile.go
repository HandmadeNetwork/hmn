package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(DeleteImageFile{})
}

type DeleteImageFile struct{}

func (m DeleteImageFile) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 7, 25, 2, 48, 1, 0, time.UTC))
}

func (m DeleteImageFile) Name() string {
	return "DeleteImageFile"
}

func (m DeleteImageFile) Description() string {
	return "Removes the imagefile table, replacing all uses with assets IDs"
}

func (m DeleteImageFile) Up(ctx context.Context, tx pgx.Tx) error {
	// NOTE(ben): Only project screenshots and podcast art uses image files.
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project_screenshot
			ADD COLUMN asset_id UUID REFERENCES asset (id) ON DELETE CASCADE;
		ALTER TABLE podcast
			ADD COLUMN image_asset UUID REFERENCES asset (id) ON DELETE SET NULL;

		UPDATE project_screenshot
		SET asset_id = image_file.asset_id
		FROM image_file
		WHERE project_screenshot.imagefile_id = image_file.id;

		UPDATE podcast
		SET image_asset = image_file.asset_id
		FROM image_file
		WHERE podcast.image_id = image_file.id;
		`,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE project_screenshot
			DROP COLUMN imagefile_id;
		ALTER TABLE podcast
			DROP COLUMN image_id;
		DROP TABLE image_file;
	`)
	if err != nil {
		return err
	}

	return nil
}

func (m DeleteImageFile) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE project_screenshot DROP COLUMN asset_id;
		ALTER TABLE podcast DROP COLUMN image_asset;

		CREATE TABLE image_file (
			id INTEGER NOT NULL PRIMARY KEY,
			file VARCHAR(255) NOT NULL,
			size INTEGER NOT NULL,
			sha1sum VARCHAR(40) NOT NULL,
			protected BOOLEAN NOT NULL,
			height INTEGER NOT NULL,
			width INTEGER NOT NULL,
			asset_id UUID REFERENCES asset (id) ON DELETE SET NULL
		);
		CREATE SEQUENCE image_file_id_seq OWNED BY image_file.id;
		ALTER TABLE image_file ALTER COLUMN id SET DEFAULT nextval('image_file_id_seq');

		ALTER TABLE project_screenshot
			ADD COLUMN imagefile_id INTEGER REFERENCES image_file (id);
		ALTER TABLE podcast
			ADD COLUMN image_id INTEGER REFERENCES image_file (id);

		CREATE INDEX ON project_screenshot (imagefile_id);
		ALTER TABLE project_screenshot
			ADD CONSTRAINT project_screenshot_project_id_imagefile_id_uniq UNIQUE (project_id, imagefile_id);

		CREATE INDEX ON podcast (image_id);
	`)
	return err
}

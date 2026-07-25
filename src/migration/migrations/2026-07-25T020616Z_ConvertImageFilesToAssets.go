package migrations

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(ConvertImageFilesToAssets{})
}

type ConvertImageFilesToAssets struct{}

func (m ConvertImageFilesToAssets) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 7, 25, 2, 6, 16, 0, time.UTC))
}

func (m ConvertImageFilesToAssets) Name() string {
	return "ConvertImageFilesToAssets"
}

func (m ConvertImageFilesToAssets) Description() string {
	return "Uploads all of the image files in the db to S3 and tracks their IDs"
}

// Copied here from `models` because, well, we're about to delete it
type ImageFile struct {
	ID        int    `db:"id"`
	File      string `db:"file"` // relative to public/media
	Size      int    `db:"size"`
	Sha1Sum   string `db:"sha1sum"`
	Protected bool   `db:"protected"`
	Height    int    `db:"height"`
	Width     int    `db:"width"`
}

func (m ConvertImageFilesToAssets) Up(ctx context.Context, tx pgx.Tx) error {
	// NOTE(ben): First delete all image_file records whose image no longer
	// exists on disk. We unfortunately have to manually cascade this into
	// project screenshots and podcasts.
	{
		files, err := db.Query[ImageFile](ctx, tx, `SELECT $columns FROM image_file`)
		if err != nil {
			return err
		}

		var missingFileIDs []int
		for _, file := range files {
			_, err := os.Stat(filepath.Join("public", "media", file.File))
			if errors.Is(err, os.ErrNotExist) {
				fmt.Printf(
					"WARNING: File %s does not exist on disk, and its image_file record will be deleted.\n",
					file.File,
				)
				missingFileIDs = append(missingFileIDs, file.ID)
			} else if err != nil {
				return err
			}
		}
		_, err1 := tx.Exec(ctx,
			`DELETE FROM project_screenshot WHERE imagefile_id = ANY($1)`,
			missingFileIDs,
		)
		_, err2 := tx.Exec(ctx,
			`UPDATE podcast SET image_id = NULL WHERE image_id = ANY($1)`,
			missingFileIDs,
		)
		_, err3 := tx.Exec(ctx,
			`DELETE FROM image_file WHERE id = ANY($1)`,
			missingFileIDs,
		)
		if err := errors.Join(err1, err2, err3); err != nil {
			return err
		}
	}

	// NOTE(ben): Upload all image files as assets. If somehow this fails and we
	// have to roll back the transaction, we will have created a few unused
	// assets. OH WELL
	{
		files, err := db.Query[ImageFile](ctx, tx, `SELECT $columns FROM image_file`)
		if err != nil {
			return err
		}
		newAssets := make(map[int]*models.Asset)
		for i, file := range files {
			fmt.Printf("Uploading %d of %d: %s...\n", i+1, len(files), file.File)
			contents, err := os.ReadFile(filepath.Join("public", "media", file.File))
			if err != nil {
				return err
			}

			asset, err := assets.Create(ctx, tx, assets.CreateInput{
				Content:  contents,
				Filename: filepath.Base(file.File),

				Width:  file.Width,
				Height: file.Height,
			})
			if err != nil {
				return err
			}

			newAssets[file.ID] = asset
		}

		_, err = tx.Exec(ctx,
			`
			ALTER TABLE image_file
				ADD COLUMN asset_id UUID REFERENCES asset (id) ON DELETE SET NULL;
			`,
		)
		if err != nil {
			return err
		}

		// NOTE(ben): Feels dumb, but we're just going to set all the new IDs using
		// one query each. Who cares.
		for fileID, asset := range newAssets {
			_, err := tx.Exec(ctx,
				`
				UPDATE image_file SET asset_id = $1 WHERE id = $2
				`,
				asset.ID, fileID,
			)
			if err != nil {
				return err
			}
		}
	}

	// NOTE(ben): Sanity check
	numNull, err := db.QueryOneScalar[int](ctx, tx, `SELECT COUNT(*) FROM image_file WHERE asset_id IS NULL`)
	if err != nil {
		return err
	}
	if numNull != 0 {
		return fmt.Errorf("expected all remaining image files to get assets")
	}

	return nil
}

func (m ConvertImageFilesToAssets) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE image_file
			DROP COLUMN asset_id;
		`,
	)
	return err
}

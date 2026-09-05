package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(ProjectGallery{})
}

type ProjectGallery struct{}

func (m ProjectGallery) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 8, 29, 2, 39, 37, 0, time.UTC))
}

func (m ProjectGallery) Name() string {
	return "ProjectGallery"
}

func (m ProjectGallery) Description() string {
	return "Add gallery fields to projects"
}

func (m ProjectGallery) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE project
			ADD COLUMN gallery BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN gallery_sort INTEGER NOT NULL DEFAULT 0,
			ADD COLUMN gallery_desc TEXT NOT NULL DEFAULT '';
	`)
	return err
}

func (m ProjectGallery) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE project
			DROP COLUMN gallery,
			DROP COLUMN gallery_sort,
			DROP COLUMN gallery_desc;
	`)
	return err
}

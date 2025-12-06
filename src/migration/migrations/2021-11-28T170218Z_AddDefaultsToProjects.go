package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddDefaultsToProjects{})
}

type AddDefaultsToProjects struct{}

func (m AddDefaultsToProjects) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 11, 28, 17, 2, 18, 0, time.UTC))
}

func (m AddDefaultsToProjects) Name() string {
	return "AddDefaultsToProjects"
}

func (m AddDefaultsToProjects) Description() string {
	return "Add default values to many project columns"
}

func (m AddDefaultsToProjects) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_project
			ALTER COLUMN slug SET DEFAULT '',
			ALTER COLUMN color_1 SET DEFAULT 'ab4c47',
			ALTER COLUMN color_2 SET DEFAULT 'a5467d',
			ALTER COLUMN featured SET DEFAULT FALSE,
			ALTER COLUMN hidden SET DEFAULT FALSE,
			ALTER COLUMN blog_enabled SET DEFAULT FALSE,
			ALTER COLUMN forum_enabled SET DEFAULT FALSE,
			ALTER COLUMN all_last_updated SET DEFAULT 'epoch',
			ALTER COLUMN annotation_last_updated SET DEFAULT 'epoch',
			ALTER COLUMN blog_last_updated SET DEFAULT 'epoch',
			ALTER COLUMN forum_last_updated SET DEFAULT 'epoch',
			ALTER COLUMN date_approved SET DEFAULT 'epoch',
			ALTER COLUMN bg_flags SET DEFAULT 0,
			ALTER COLUMN library_enabled SET DEFAULT FALSE;
		`,
	)
	return err
}

func (m AddDefaultsToProjects) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_project
			ALTER COLUMN slug DROP DEFAULT,
			ALTER COLUMN color_1 DROP DEFAULT,
			ALTER COLUMN color_2 DROP DEFAULT,
			ALTER COLUMN featured DROP DEFAULT,
			ALTER COLUMN hidden DROP DEFAULT,
			ALTER COLUMN blog_enabled DROP DEFAULT,
			ALTER COLUMN forum_enabled DROP DEFAULT,
			ALTER COLUMN all_last_updated DROP DEFAULT,
			ALTER COLUMN annotation_last_updated DROP DEFAULT,
			ALTER COLUMN blog_last_updated DROP DEFAULT,
			ALTER COLUMN forum_last_updated DROP DEFAULT,
			ALTER COLUMN date_approved DROP DEFAULT,
			ALTER COLUMN bg_flags DROP DEFAULT,
			ALTER COLUMN library_enabled DROP DEFAULT;
		`,
	)
	return err
}

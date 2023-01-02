package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddJamProjects{})
}

type AddJamProjects struct{}

func (m AddJamProjects) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 6, 18, 1, 3, 39, 0, time.UTC))
}

func (m AddJamProjects) Name() string {
	return "AddJamProjects"
}

func (m AddJamProjects) Description() string {
	return "Add jam and project association table"
}

func (m AddJamProjects) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE jam_project (
			project_id INT REFERENCES project (id) ON DELETE CASCADE,
			jam_slug VARCHAR(64) NOT NULL,
			participating BOOLEAN NOT NULL DEFAULT FALSE,
			UNIQUE (project_id, jam_slug)
		);
		CREATE INDEX jam_project_jam_slug ON jam_project (jam_slug);
		CREATE INDEX jam_project_project_id ON jam_project (project_id);
		`,
	)
	return err
}

func (m AddJamProjects) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP INDEX jam_project_jam_slug;
		DROP INDEX jam_project_project_id;
		DROP TABLE jam_project;
		`,
	)
	return err
}

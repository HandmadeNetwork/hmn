package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(CleanUpProjects{})
}

type CleanUpProjects struct{}

func (m CleanUpProjects) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 11, 6, 3, 38, 35, 0, time.UTC))
}

func (m CleanUpProjects) Name() string {
	return "CleanUpProjects"
}

func (m CleanUpProjects) Description() string {
	return "Clean up projects with data violating our new constraints"
}

func (m CleanUpProjects) Up(ctx context.Context, tx pgx.Tx) error {
	var err error

	_, err = tx.Exec(ctx, `
		DELETE FROM handmade_project WHERE id IN (91, 92);
		DELETE FROM handmade_communicationchoicelist
		WHERE project_id IN (91, 92);
		DELETE FROM handmade_project_languages
		WHERE project_id IN (91, 92);
		DELETE FROM handmade_project_screenshots
		WHERE project_id IN (91, 92);
		
		UPDATE handmade_project
		SET slug = 'hmh-notes'
		WHERE slug = 'hmh_notes';
	`)
	if err != nil {
		return oops.New(err, "failed to patch up project slugs")
	}

	return nil
}

func (m CleanUpProjects) Down(ctx context.Context, tx pgx.Tx) error {
	var err error

	_, err = tx.Exec(ctx, `
		-- Don't bother restoring those old projects

		UPDATE handmade_project
		SET slug = 'hmh_notes'
		WHERE slug = 'hmh-notes';
	`)
	if err != nil {
		return oops.New(err, "failed to restore project slugs")
	}

	return nil
}

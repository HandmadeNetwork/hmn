package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(PersonalProjects{})
}

type PersonalProjects struct{}

func (m PersonalProjects) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 11, 6, 3, 39, 30, 0, time.UTC))
}

func (m PersonalProjects) Name() string {
	return "PersonalProjects"
}

func (m PersonalProjects) Description() string {
	return "Add data model for personal projects / tags"
}

func (m PersonalProjects) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_project
			DROP featurevotes,
			DROP parent_id,
			DROP quota,
			DROP quota_used,
			DROP standalone,
			ALTER flags TYPE BOOLEAN USING flags > 0;
		
		ALTER TABLE handmade_project RENAME flags TO hidden;
	`)
	if err != nil {
		return oops.New(err, "failed to clean up existing fields")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project
			ADD personal BOOLEAN NOT NULL DEFAULT TRUE,
			ADD tag VARCHAR(20) NOT NULL DEFAULT '';
	`)
	if err != nil {
		return oops.New(err, "failed to add new fields")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project	
			ADD CONSTRAINT slug_syntax CHECK (
				slug ~ '^([a-z0-9]+(-[a-z0-9]+)*)?$'
			),
			ADD CONSTRAINT tag_syntax CHECK (
				tag ~ '^([a-z0-9]+(-[a-z0-9]+)*)?$'
			)
	`)
	if err != nil {
		return oops.New(err, "failed to add check constraints")
	}

	_, err = tx.Exec(ctx, `
		UPDATE handmade_project
		SET personal = FALSE;
	`)
	if err != nil {
		return oops.New(err, "failed to make existing projects official")
	}

	return nil
}

func (m PersonalProjects) Down(ctx context.Context, tx pgx.Tx) error {
	var err error

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project
			DROP CONSTRAINT slug_syntax,
			DROP CONSTRAINT tag_syntax,
			DROP personal,
			DROP tag,
			ADD featurevotes INT NOT NULL DEFAULT 0,
			-- no projects actually have a parent id so thankfully no further updates to do
			ADD parent_id INT REFERENCES handmade_project (id) ON DELETE SET NULL,
			ADD quota INT NOT NULL DEFAULT 0,
			ADD quota_used INT NOT NULL DEFAULT 0,
			ADD standalone BOOLEAN NOT NULL DEFAULT FALSE,
			ALTER hidden TYPE INT USING CASE WHEN hidden THEN 1 ELSE 0 END;

		ALTER TABLE handmade_project RENAME hidden TO flags;
	`)
	if err != nil {
		return oops.New(err, "failed to revert personal project changes")
	}

	return nil
}

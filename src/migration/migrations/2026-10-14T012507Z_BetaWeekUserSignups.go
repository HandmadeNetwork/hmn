package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(BetaWeekUserSignups{})
}

type BetaWeekUserSignups struct{}

func (m BetaWeekUserSignups) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 10, 14, 1, 25, 7, 0, time.UTC))
}

func (m BetaWeekUserSignups) Name() string {
	return "BetaWeekUserSignups"
}

func (m BetaWeekUserSignups) Description() string {
	return "Add general beta tester signup data"
}

func (m BetaWeekUserSignups) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE betaweek_project_submission
			ADD COLUMN approved BOOLEAN NOT NULL DEFAULT FALSE,
			ADD COLUMN description TEXT NOT NULL DEFAULT '',
			ADD COLUMN platform_list TEXT[] NOT NULL DEFAULT '{}',
			ADD COLUMN sort INTEGER NOT NULL DEFAULT 0;

		CREATE TABLE betaweek_testing_signup (
			project_id INTEGER NOT NULL REFERENCES project,
			user_id INTEGER NOT NULL REFERENCES hmn_user,
			PRIMARY KEY (project_id, user_id)
		)
	`)
	return err
}

func (m BetaWeekUserSignups) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		DROP TABLE betaweek_testing_signup;

		ALTER TABLE betaweek_project_submission
			DROP COLUMN approved,
			DROP COLUMN description,
			DROP COLUMN platform_list,
			DROP COLUMN sort;
	`)
	return err
}

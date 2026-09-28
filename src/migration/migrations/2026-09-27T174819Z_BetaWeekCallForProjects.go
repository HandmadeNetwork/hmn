package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(BetaWeekCallForProjects{})
}

type BetaWeekCallForProjects struct{}

func (m BetaWeekCallForProjects) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 9, 27, 17, 48, 19, 0, time.UTC))
}

func (m BetaWeekCallForProjects) Name() string {
	return "BetaWeekCallForProjects"
}

func (m BetaWeekCallForProjects) Description() string {
	return "Add a table for beta week submissions"
}

func (m BetaWeekCallForProjects) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE betaweek_project_submission (
			id SERIAL NOT NULL PRIMARY KEY,
			event_slug VARCHAR(64) NOT NULL,

			project_id INT NOT NULL REFERENCES project,
			project_status TEXT NOT NULL,
			project_platforms TEXT NOT NULL,
			project_instructions TEXT NOT NULL,
			author_availability TEXT NOT NULL,
			author_goals TEXT NOT NULL
		)
		`,
	)
	return err
}

func (m BetaWeekCallForProjects) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP TABLE betaweek_project_submission;
		`,
	)
	return err
}

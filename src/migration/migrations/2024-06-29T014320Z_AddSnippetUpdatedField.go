package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddSnippetUpdatedField{})
}

type AddSnippetUpdatedField struct{}

func (m AddSnippetUpdatedField) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2024, 6, 29, 1, 43, 20, 0, time.UTC))
}

func (m AddSnippetUpdatedField) Name() string {
	return "AddSnippetUpdatedField"
}

func (m AddSnippetUpdatedField) Description() string {
	return "Add field to track most recent snippets on projects"
}

func (m AddSnippetUpdatedField) Up(ctx context.Context, tx pgx.Tx) error {
	utils.Must1(tx.Exec(ctx,
		`
		ALTER TABLE project
			ADD COLUMN snippet_last_posted TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT 'epoch';
		`,
	))
	utils.Must(hmndata.UpdateSnippetLastPostedForAllProjects(ctx, tx))
	return nil
}

func (m AddSnippetUpdatedField) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE project
			DROP COLUMN snippet_last_posted;
		`,
	)
	return err
}

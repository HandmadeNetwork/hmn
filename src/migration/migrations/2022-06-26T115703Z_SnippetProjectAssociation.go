package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(SnippetProjectAssociation{})
}

type SnippetProjectAssociation struct{}

func (m SnippetProjectAssociation) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2022, 6, 26, 11, 57, 3, 0, time.UTC))
}

func (m SnippetProjectAssociation) Name() string {
	return "SnippetProjectAssociation"
}

func (m SnippetProjectAssociation) Description() string {
	return "Table for associating a snippet with projects"
}

func (m SnippetProjectAssociation) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		CREATE TABLE snippet_project (
			snippet_id INTEGER NOT NULL REFERENCES snippet (id) ON DELETE CASCADE,
			project_id INTEGER NOT NULL REFERENCES project (id) ON DELETE CASCADE,
			kind INTEGER NOT NULL,
			UNIQUE (snippet_id, project_id)
		);
		`,
	)
	return err
}

func (m SnippetProjectAssociation) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		DROP TABLE snippet_project;
		`,
	)
	return err
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddJamSnippetField{})
}

type AddJamSnippetField struct{}

func (m AddJamSnippetField) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 9, 25, 2, 38, 10, 0, time.UTC))
}

func (m AddJamSnippetField) Name() string {
	return "AddJamSnippetField"
}

func (m AddJamSnippetField) Description() string {
	return "Add a special field for jam snippets"
}

func (m AddJamSnippetField) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_snippet
			ADD is_jam BOOLEAN NOT NULL DEFAULT FALSE;
	`)
	if err != nil {
		return oops.New(err, "failed to add jam column")
	}

	return nil
}

func (m AddJamSnippetField) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_snippet
			DROP is_jam;
	`)
	if err != nil {
		return oops.New(err, "failed to drop jam column")
	}

	return nil
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddHTMLPreview{})
}

type AddHTMLPreview struct{}

func (m AddHTMLPreview) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 3, 6, 17, 9, 58, 0, time.UTC))
}

func (m AddHTMLPreview) Name() string {
	return "AddHTMLPreview"
}

func (m AddHTMLPreview) Description() string {
	return "Adds an explicit HTML preview to post versions"
}

func (m AddHTMLPreview) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE post
			ADD COLUMN preview_html TEXT NOT NULL DEFAULT '';
		`,
	)
	return err
}

func (m AddHTMLPreview) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE post
			DROP COLUMN preview_html;
		`,
	)
	return err
}

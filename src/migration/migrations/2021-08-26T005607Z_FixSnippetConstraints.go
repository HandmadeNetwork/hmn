package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(FixSnippetConstraints{})
}

type FixSnippetConstraints struct{}

func (m FixSnippetConstraints) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 26, 0, 56, 7, 0, time.UTC))
}

func (m FixSnippetConstraints) Name() string {
	return "FixSnippetConstraints"
}

func (m FixSnippetConstraints) Description() string {
	return "Fix the ON DELETE behaviors of snippets"
}

func (m FixSnippetConstraints) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_snippet
			DROP CONSTRAINT handmade_snippet_asset_id_c786de4f_fk_handmade_asset_id,
			DROP CONSTRAINT handmade_snippet_discord_message_id_d16f1f4e_fk_handmade_,
			DROP CONSTRAINT handmade_snippet_owner_id_fcca1783_fk_auth_user_id,
			ADD FOREIGN KEY (asset_id) REFERENCES handmade_asset (id) ON DELETE SET NULL,
			ADD FOREIGN KEY (discord_message_id) REFERENCES handmade_discordmessage (id) ON DELETE SET NULL,
			ADD FOREIGN KEY (owner_id) REFERENCES auth_user (id) ON DELETE CASCADE;
	`)
	if err != nil {
		return oops.New(err, "failed to fix constraints")
	}

	return nil
}

func (m FixSnippetConstraints) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

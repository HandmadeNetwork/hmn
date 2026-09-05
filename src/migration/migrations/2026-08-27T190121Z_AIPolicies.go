package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AIPolicies{})
}

type AIPolicies struct{}

func (m AIPolicies) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2026, 8, 27, 19, 1, 21, 0, time.UTC))
}

func (m AIPolicies) Name() string {
	return "AIPolicies"
}

func (m AIPolicies) Description() string {
	return "Add AI policies to projects"
}

func (m AIPolicies) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE project
			ADD COLUMN ai_policy TEXT NOT NULL DEFAULT '',
			ADD COLUMN ai_policy_parsed TEXT NOT NULL DEFAULT '';
	`)
	return err
}

func (m AIPolicies) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE project
			DROP COLUMN ai_policy,
			DROP COLUMN ai_policy_parsed;
	`)
	return err
}

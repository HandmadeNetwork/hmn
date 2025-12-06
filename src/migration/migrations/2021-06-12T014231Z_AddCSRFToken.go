package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddCSRFToken{})
}

type AddCSRFToken struct{}

func (m AddCSRFToken) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 6, 12, 1, 42, 31, 0, time.UTC))
}

func (m AddCSRFToken) Name() string {
	return "AddCSRFToken"
}

func (m AddCSRFToken) Description() string {
	return "Adds a CSRF token to user sessions"
}

func (m AddCSRFToken) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE sessions
			ADD csrf_token VARCHAR(30) NOT NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to add CSRF token column")
	}

	return nil
}

func (m AddCSRFToken) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

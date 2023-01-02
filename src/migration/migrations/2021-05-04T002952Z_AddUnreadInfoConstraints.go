package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddUnreadInfoConstraints{})
}

type AddUnreadInfoConstraints struct{}

func (m AddUnreadInfoConstraints) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 5, 4, 0, 29, 52, 0, time.UTC))
}

func (m AddUnreadInfoConstraints) Name() string {
	return "AddUnreadInfoConstraints"
}

func (m AddUnreadInfoConstraints) Description() string {
	return "Add more constraints to the unread info tables"
}

func (m AddUnreadInfoConstraints) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_threadlastreadinfo
			ALTER lastread SET NOT NULL,
			ALTER thread_id SET NOT NULL,
			DROP category_id,
			ADD UNIQUE (thread_id, user_id);
		
		ALTER TABLE handmade_categorylastreadinfo
			ALTER lastread SET NOT NULL,
			ALTER category_id SET NOT NULL,
			ADD UNIQUE (category_id, user_id);
	`)
	if err != nil {
		return oops.New(err, "failed to add constraints")
	}

	return nil
}

func (m AddUnreadInfoConstraints) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(NewLinkData{})
}

type NewLinkData struct{}

func (m NewLinkData) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 27, 19, 4, 8, 0, time.UTC))
}

func (m NewLinkData) Name() string {
	return "NewLinkData"
}

func (m NewLinkData) Description() string {
	return "Rework link data to be less completely weird"
}

func (m NewLinkData) Up(ctx context.Context, tx pgx.Tx) error {
	/*
		Broadly the goal is to:
		- drop `key`
		- make `name` not null
		- rename `value` to `url`
	*/

	_, err := tx.Exec(ctx, `UPDATE handmade_links SET name = '' WHERE name IS NULL`)
	if err != nil {
		return oops.New(err, "failed to fill in null names")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_links
			DROP key,
			ALTER name SET NOT NULL;

		ALTER TABLE handmade_links
			RENAME value TO url;
	`)
	if err != nil {
		return oops.New(err, "failed to alter links table")
	}

	return nil
}

func (m NewLinkData) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(ChangeHMNColors{})
}

type ChangeHMNColors struct{}

func (m ChangeHMNColors) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 8, 28, 12, 12, 28, 0, time.UTC))
}

func (m ChangeHMNColors) Name() string {
	return "ChangeHMNColors"
}

func (m ChangeHMNColors) Description() string {
	return "Change the colors for the HMN project"
}

func (m ChangeHMNColors) Up(ctx context.Context, tx pgx.Tx) error {
	tag, err := tx.Exec(ctx, `
		UPDATE handmade_project
		SET
			color_1 = 'ab4c47',
			color_2 = 'a5467d'
		WHERE
			slug = 'hmn'
	`)
	if err != nil {
		return oops.New(err, "failed to update HMN colors")
	}

	if tag.RowsAffected() != 1 {
		return oops.New(nil, "was supposed to update only HMN, but updated %d projects instead", tag.RowsAffected())
	}

	return nil
}

func (m ChangeHMNColors) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

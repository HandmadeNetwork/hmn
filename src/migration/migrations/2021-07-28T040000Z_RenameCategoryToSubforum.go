package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(RenameCategoryToSubforum{})
}

type RenameCategoryToSubforum struct{}

func (m RenameCategoryToSubforum) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 28, 4, 0, 0, 0, time.UTC))
}

func (m RenameCategoryToSubforum) Name() string {
	return "RenameCategoryToSubforum"
}

func (m RenameCategoryToSubforum) Description() string {
	return "Rename categories to subforums"
}

func (m RenameCategoryToSubforum) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_category
			RENAME TO handmade_subforum;

		ALTER TABLE handmade_subforum
			ALTER project_id SET NOT NULL,
			ALTER slug SET NOT NULL,
			ALTER name SET NOT NULL,
			ALTER blurb SET NOT NULL,
			ALTER blurb SET DEFAULT '',
			DROP kind,
			DROP depth,
			DROP color_1,
			DROP color_2;
		
		ALTER TABLE handmade_categorylastreadinfo
			RENAME TO handmade_subforumlastreadinfo;
		ALTER TABLE handmade_subforumlastreadinfo
			RENAME category_id TO subforum_id;
	`)
	if err != nil {
		return oops.New(err, "failed to rename stuff")
	}

	return nil
}

func (m RenameCategoryToSubforum) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

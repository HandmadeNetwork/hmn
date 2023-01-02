package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(RemovePostText{})
}

type RemovePostText struct{}

func (m RemovePostText) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 23, 2, 11, 47, 0, time.UTC))
}

func (m RemovePostText) Name() string {
	return "RemovePostText"
}

func (m RemovePostText) Description() string {
	return "Collapse handmade_posttext and handmade_posttextversion into one table"
}

func (m RemovePostText) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		CREATE TABLE handmade_postversion (
			id INT PRIMARY KEY,
			post_id INT NOT NULL REFERENCES handmade_post(id) ON DELETE CASCADE,
			
			text_raw TEXT NOT NULL,
			text_parsed TEXT NOT NULL,
			parser INT NOT NULL,

			edit_ip INET,
			edit_date TIMESTAMP WITH TIME ZONE NOT NULL,
			edit_reason VARCHAR(255) NOT NULL DEFAULT '',
			editor_id INT REFERENCES auth_user(id) ON DELETE SET NULL
		);
	`)
	if err != nil {
		return oops.New(err, "failed to create new postversion table")
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO handmade_postversion
		SELECT
			tver.id,
			tver.post_id,

			t.text,
			t.textparsed,
			t.parser,

			tver.editip,
			COALESCE(tver.editdate, p.postdate),
			COALESCE(tver.editreason, ''),
			tver.editor_id
		FROM
			handmade_posttextversion AS tver
			JOIN handmade_posttext AS t ON tver.text_id = t.id
			JOIN handmade_post AS p ON tver.post_id = p.id
		WHERE
			tver.post_id IS NOT NULL
	`)
	if err != nil {
		return oops.New(err, "failed to create postversions")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_post
			DROP CONSTRAINT handmade_post_current_id_762211b7_fk_handmade_,
			ADD FOREIGN KEY (current_id) REFERENCES handmade_postversion ON DELETE RESTRICT;
	`)
	if err != nil {
		return oops.New(err, "failed to drop old current constraint")
	}

	_, err = tx.Exec(ctx, `
		DROP TABLE handmade_posttextversion;
		DROP TABLE handmade_posttext;
	`)
	if err != nil {
		return oops.New(err, "failed to drop tables")
	}

	return nil
}

func (m RemovePostText) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

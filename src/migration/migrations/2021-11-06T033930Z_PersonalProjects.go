package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/db"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(PersonalProjects{})
}

type PersonalProjects struct{}

func (m PersonalProjects) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 11, 6, 3, 39, 30, 0, time.UTC))
}

func (m PersonalProjects) Name() string {
	return "PersonalProjects"
}

func (m PersonalProjects) Description() string {
	return "Add data model for personal projects / tags"
}

func (m PersonalProjects) Up(ctx context.Context, tx pgx.Tx) error {
	var err error

	_, err = tx.Exec(ctx, `
		CREATE TABLE tags (
			id SERIAL NOT NULL PRIMARY KEY,
			text VARCHAR(20) NOT NULL
		);
		CREATE INDEX tags_by_text ON tags (text);

		ALTER TABLE tags
			ADD CONSTRAINT tag_syntax CHECK (
				text ~ '^([a-z0-9]+(-[a-z0-9]+)*)?$'
			);

		CREATE TABLE snippet_tags (
			snippet_id INT NOT NULL REFERENCES handmade_snippet (id) ON DELETE CASCADE,
			tag_id INT NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
			PRIMARY KEY (snippet_id, tag_id)
		);
	`)
	if err != nil {
		return oops.New(err, "failed to add tags tables")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_snippet
			DROP CONSTRAINT handmade_snippet_owner_id_fkey,
			ALTER owner_id DROP NOT NULL,
			ADD CONSTRAINT handmade_snippet_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES auth_user (id) ON DELETE SET NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to update snippet constraints")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project
			DROP featurevotes,
			DROP parent_id,
			DROP quota,
			DROP quota_used,
			DROP standalone,
			ALTER flags TYPE BOOLEAN USING flags > 0;
		
		ALTER TABLE handmade_project RENAME flags TO hidden;
	`)
	if err != nil {
		return oops.New(err, "failed to clean up existing fields")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project
			ADD personal BOOLEAN NOT NULL DEFAULT TRUE,
			ADD tag INT REFERENCES tags (id);
	`)
	if err != nil {
		return oops.New(err, "failed to add new fields")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project	
			ADD CONSTRAINT slug_syntax CHECK (
				slug ~ '^([a-z0-9]+(-[a-z0-9]+)*)?$'
			);
	`)
	if err != nil {
		return oops.New(err, "failed to add check constraints")
	}

	_, err = tx.Exec(ctx, `
		UPDATE handmade_project
		SET personal = FALSE;
	`)
	if err != nil {
		return oops.New(err, "failed to make existing projects official")
	}

	//
	// Port "jam snippets" to use a tag
	//

	jamTagId, err := db.QueryInt(ctx, tx, `INSERT INTO tags (text) VALUES ('wheeljam') RETURNING id`)
	if err != nil {
		return oops.New(err, "failed to create jam tag")
	}

	_, err = tx.Exec(ctx,
		`
		INSERT INTO snippet_tags
			SELECT id, $1
			FROM handmade_snippet
			WHERE is_jam
		`,
		jamTagId,
	)
	if err != nil {
		return oops.New(err, "failed to add jam tag to jam snippets")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_snippet
			DROP is_jam;
	`)
	if err != nil {
		return oops.New(err, "failed to drop is_jam column from snippets")
	}

	return nil
}

func (m PersonalProjects) Down(ctx context.Context, tx pgx.Tx) error {
	var err error

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_snippet
			ADD is_jam BOOLEAN NOT NULL DEFAULT FALSE;

		UPDATE handmade_snippet
		SET is_jam = TRUE
		WHERE id IN (
			SELECT snippet.id
			FROM
				handmade_snippet AS snippet
				JOIN snippet_tags ON snippet.id = snippet_tags.snippet_id
				JOIN tags ON snippet_tags.tag_id = tags.id
			WHERE
				tags.text = 'wheeljam'
		);

		DELETE FROM tags WHERE text = 'wheeljam';
	`)
	if err != nil {
		return oops.New(err, "failed to revert jam snippets")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project
			DROP CONSTRAINT slug_syntax,
			DROP personal,
			DROP tag,
			ADD featurevotes INT NOT NULL DEFAULT 0,
			-- no projects actually have a parent id so thankfully no further updates to do
			ADD parent_id INT REFERENCES handmade_project (id) ON DELETE SET NULL,
			ADD quota INT NOT NULL DEFAULT 0,
			ADD quota_used INT NOT NULL DEFAULT 0,
			ADD standalone BOOLEAN NOT NULL DEFAULT FALSE,
			ALTER hidden TYPE INT USING CASE WHEN hidden THEN 1 ELSE 0 END;

		ALTER TABLE handmade_project RENAME hidden TO flags;
	`)
	if err != nil {
		return oops.New(err, "failed to revert personal project changes")
	}

	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_snippet
			DROP CONSTRAINT handmade_snippet_owner_id_fkey,
			ALTER owner_id SET NOT NULL,
			ADD CONSTRAINT handmade_snippet_owner_id_fkey FOREIGN KEY (owner_id) REFERENCES auth_user (id) ON DELETE CASCADE;
	`)
	if err != nil {
		return oops.New(err, "failed to revert snippet constraint changes")
	}

	_, err = tx.Exec(ctx, `
		DROP TABLE snippet_tags;		
		DROP TABLE tags;
	`)
	if err != nil {
		return oops.New(err, "failed to drop tags table")
	}

	return nil
}

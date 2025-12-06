package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddCommonFieldsToPosts{})
}

type AddCommonFieldsToPosts struct{}

func (m AddCommonFieldsToPosts) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 26, 23, 57, 20, 0, time.UTC))
}

func (m AddCommonFieldsToPosts) Name() string {
	return "AddCommonFieldsToPosts"
}

func (m AddCommonFieldsToPosts) Description() string {
	return "Adds project and category info directly to posts for more efficient queries"
}

func (m AddCommonFieldsToPosts) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_post
			ADD category_kind INT,
			ADD project_id INT REFERENCES handmade_project (id) ON DELETE RESTRICT;
		`,
	)
	if err != nil {
		return oops.New(err, "failed to add columns")
	}

	_, err = tx.Exec(ctx,
		`
		UPDATE handmade_post
		SET (category_id, category_kind, project_id) = (
			SELECT cat.id, cat.kind, cat.project_id
			FROM
				handmade_category AS cat
				JOIN handmade_thread AS thread ON thread.category_id = cat.id
				JOIN handmade_post AS post ON post.thread_id = thread.id
			WHERE
				post.id = handmade_post.id
		)
		`,
	)
	if err != nil {
		return oops.New(err, "failed to migrate data from categories")
	}

	_, err = tx.Exec(ctx,
		`
		CREATE FUNCTION category_id_for_thread(int) returns int as $$
			SELECT thread.category_id
			FROM handmade_thread AS thread
			WHERE thread.id = $1
		$$ LANGUAGE SQL;

		CREATE FUNCTION category_kind_for_post(int) returns int as $$
			SELECT cat.kind
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
				JOIN handmade_category AS cat ON thread.category_id = cat.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;

		CREATE FUNCTION project_id_for_post(int) returns int as $$
			SELECT cat.project_id
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
				JOIN handmade_category AS cat ON thread.category_id = cat.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;

		ALTER TABLE handmade_post
			ALTER category_kind SET NOT NULL,
			ALTER thread_id SET NOT NULL,
			ALTER project_id SET NOT NULL,
			ADD CONSTRAINT post_category_id_from_thread CHECK (
				category_id_for_thread(thread_id) = category_id
			),
			ADD CONSTRAINT post_category_kind_from_category CHECK (
				category_kind_for_post(id) = category_kind
			),
			ADD CONSTRAINT post_project_id_from_category CHECK (
				project_id_for_post(id) = project_id
			);
		`,
	)
	if err != nil {
		return oops.New(err, "failed to add constraints")
	}

	_, err = tx.Exec(ctx,
		`
		CREATE INDEX post_project_id ON handmade_post (project_id);
		CREATE INDEX post_category_kind ON handmade_post (category_kind);
		CREATE INDEX post_postdate ON handmade_post (postdate DESC);

		CREATE INDEX clri_user_id ON handmade_categorylastreadinfo (user_id);
		CREATE INDEX tlri_user_id ON handmade_threadlastreadinfo (user_id);
		`,
	)
	if err != nil {
		return oops.New(err, "failed to create indexes")
	}

	return nil
}

func (m AddCommonFieldsToPosts) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

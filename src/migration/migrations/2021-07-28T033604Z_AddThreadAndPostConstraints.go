package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddThreadAndPostConstraints{})
}

type AddThreadAndPostConstraints struct{}

func (m AddThreadAndPostConstraints) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 28, 3, 36, 4, 0, time.UTC))
}

func (m AddThreadAndPostConstraints) Name() string {
	return "AddThreadAndPostConstraints"
}

func (m AddThreadAndPostConstraints) Description() string {
	return "Add back appropriate check constraints for the new thread model"
}

func (m AddThreadAndPostConstraints) Up(ctx context.Context, tx pgx.Tx) error {
	// create null check constraints for threads
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_thread
			ADD CONSTRAINT thread_has_field_for_type CHECK (
				CASE
					WHEN type = 1 THEN
						subforum_id IS NULL
						AND personal_article_user_id IS NULL
					WHEN type = 2 THEN
						subforum_id IS NOT NULL
						AND personal_article_user_id IS NULL
					WHEN type = 7 THEN
						subforum_id IS NULL
						AND personal_article_user_id IS NOT NULL
					ELSE TRUE
				END
			);
	`)
	if err != nil {
		return oops.New(err, "failed to add constraint to threads")
	}

	// add constraints to posts
	_, err = tx.Exec(ctx, `
		CREATE FUNCTION thread_type_for_post(int) RETURNS int AS $$
			SELECT thread.type
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;

		CREATE FUNCTION project_id_for_post(int) RETURNS int AS $$
			SELECT thread.project_id
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;

		ALTER TABLE handmade_post
			ADD CONSTRAINT post_thread_type_from_thread CHECK (
				thread_type_for_post(id) = thread_type
			),
			ADD CONSTRAINT post_project_id_from_thread CHECK (
				project_id_for_post(id) = project_id
			);
	`)
	if err != nil {
		return oops.New(err, "failed to add post constraints")
	}

	return nil
}

func (m AddThreadAndPostConstraints) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

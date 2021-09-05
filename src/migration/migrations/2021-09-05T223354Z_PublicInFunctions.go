package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(PublicInFunctions{})
}

type PublicInFunctions struct{}

func (m PublicInFunctions) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 9, 5, 22, 33, 54, 0, time.UTC))
}

func (m PublicInFunctions) Name() string {
	return "PublicInFunctions"
}

func (m PublicInFunctions) Description() string {
	return "Make sure to put the schema in front of everything inside our postgres functions"
}

func (m PublicInFunctions) Up(ctx context.Context, tx pgx.Tx) error {
	/*
		This is due to a really stupid behavior of Postgres where, when restoring
		data, it apparently forgets that `tablename` is generally equivalent to
		`public.tablename`. This makes it totally impossible for us to actually
		restore our database backups, since these functions fail when adding new
		rows.

		https://www.postgresql.org/message-id/20180825020418.GA7869%40momjian.us
	*/

	_, err := tx.Exec(ctx, `
		CREATE OR REPLACE FUNCTION thread_type_for_post(int) RETURNS int AS $$
			SELECT thread.type
			FROM
				public.handmade_post AS post
				JOIN public.handmade_thread AS thread ON post.thread_id = thread.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;

		CREATE OR REPLACE FUNCTION project_id_for_post(int) RETURNS int AS $$
			SELECT thread.project_id
			FROM
				public.handmade_post AS post
				JOIN public.handmade_thread AS thread ON post.thread_id = thread.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;
	`)
	if err != nil {
		return oops.New(err, "failed to add post constraints")
	}

	return nil
}

func (m PublicInFunctions) Down(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `
		CREATE OR REPLACE FUNCTION thread_type_for_post(int) RETURNS int AS $$
			SELECT thread.type
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;

		CREATE OR REPLACE FUNCTION project_id_for_post(int) RETURNS int AS $$
			SELECT thread.project_id
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
			WHERE post.id = $1
		$$ LANGUAGE SQL;
	`)
	if err != nil {
		return oops.New(err, "failed to add post constraints")
	}

	return nil
}

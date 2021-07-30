package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(ReworkThreads{})
}

type ReworkThreads struct{}

func (m ReworkThreads) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 28, 2, 0, 0, 0, time.UTC))
}

func (m ReworkThreads) Name() string {
	return "ReworkThreads"
}

func (m ReworkThreads) Description() string {
	return "Detach threads from categories and make them more independent"
}

func (m ReworkThreads) Up(ctx context.Context, tx pgx.Tx) error {
	// add and rename columns
	_, err := tx.Exec(ctx, `
		ALTER TABLE handmade_thread
			ADD type INT,
			ADD project_id INT REFERENCES handmade_project (id) ON DELETE RESTRICT, -- used to associate project articles
			ALTER category_id DROP NOT NULL,
			ADD personal_article_user_id INT REFERENCES auth_user (id) ON DELETE RESTRICT; -- used to associate personal articles
		ALTER TABLE handmade_thread
			RENAME category_id TO subforum_id; -- preemptive, we're renaming categories next

		ALTER TABLE handmade_post
			RENAME category_kind TO thread_type;
		ALTER TABLE handmade_post
			DROP category_id,
			DROP CONSTRAINT post_category_kind_from_category,
			DROP CONSTRAINT post_project_id_from_category;
		
		DROP FUNCTION category_id_for_thread(int);
		DROP FUNCTION category_kind_for_post(int);
		DROP FUNCTION project_id_for_post(int);
	`)
	if err != nil {
		return oops.New(err, "failed to add and rename columns")
	}

	// fill out null thread fields
	_, err = tx.Exec(ctx, `
		UPDATE handmade_thread AS thread
		SET (type, project_id, subforum_id) = (
			SELECT kind, project_id, CASE WHEN cat.kind = 2 THEN cat.id ELSE NULL END
			FROM handmade_category AS cat
			WHERE cat.id = thread.subforum_id
		);

		ALTER TABLE handmade_thread
			ALTER type SET NOT NULL,
			ALTER project_id SET NOT NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to copy category kind to thread type")
	}

	// move wiki posts to personal articles
	_, err = tx.Exec(ctx, `
		-- turn wiki threads into personal articles
		UPDATE handmade_thread
		SET
			type = 7, -- new "personal article" type
			personal_article_user_id = 1979 -- assign to Ben for now
		WHERE type = 5;

		-- update the denormalized field on posts
		UPDATE handmade_post
		SET thread_type = 7
		WHERE thread_type = 5;
	`)
	if err != nil {
		return oops.New(err, "failed to turn wiki posts into personal articles")
	}

	// delete talk pages
	_, err = tx.Exec(ctx, `
		DELETE FROM handmade_post
		WHERE
			thread_type = 7 -- personal articles, see above
			AND parent_id IS NOT NULL;
		
		UPDATE handmade_thread
		SET last_id = first_id
		WHERE type = 7;
	`)
	if err != nil {
		return oops.New(err, "failed to delete wiki talk pages")
	}

	// delete library discussions
	_, err = tx.Exec(ctx, `
		DELETE FROM handmade_threadlastreadinfo
		WHERE thread_id IN (
			SELECT id
			FROM handmade_thread
			WHERE type = 6
		);

		DELETE FROM handmade_thread
		WHERE type = 6;

		DELETE FROM handmade_post
		WHERE thread_type = 6;

		ALTER TABLE handmade_libraryresource
			DROP category_id;
	`)
	if err != nil {
		return oops.New(err, "failed to delete library discussions")
	}

	// delete references to weirdo categories
	_, err = tx.Exec(ctx, `
		ALTER TABLE handmade_project
			DROP blog_id,
			DROP annotation_id,
			DROP wiki_id;
	`)
	if err != nil {
		return oops.New(err, "failed to delete references to categories from projects")
	}

	// delete categories we no longer need
	_, err = tx.Exec(ctx, `
		DELETE FROM handmade_categorylastreadinfo
		WHERE category_id IN (
			SELECT id
			FROM handmade_category
			WHERE kind != 2
		);

		DELETE FROM handmade_category
		WHERE kind != 2;
	`)
	if err != nil {
		return oops.New(err, "failed to delete categories")
	}

	return nil
}

func (m ReworkThreads) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

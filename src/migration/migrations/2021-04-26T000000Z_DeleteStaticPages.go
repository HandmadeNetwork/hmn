package migrations

import (
	"context"
	"fmt"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(DeleteStaticPages{})
}

type DeleteStaticPages struct{}

func (m DeleteStaticPages) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 26, 0, 0, 0, 0, time.UTC))
}

func (m DeleteStaticPages) Name() string {
	return "DeleteStaticPages"
}

func (m DeleteStaticPages) Description() string {
	return "Delete static page posts"
}

func (m DeleteStaticPages) Up(ctx context.Context, tx pgx.Tx) error {
	res, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_project
			DROP static_id;
		`,
	)
	if err != nil {
		return oops.New(err, "failed to drop static_id column")
	}

	res, err = tx.Exec(ctx,
		`
		DELETE FROM handmade_post
		WHERE id IN (
			SELECT post.id
			FROM
				handmade_post AS post
				LEFT JOIN handmade_thread AS thread ON post.thread_id = thread.id
				LEFT JOIN handmade_category AS threadcat ON thread.category_id = threadcat.id
				LEFT JOIN handmade_category AS postcat ON post.category_id = postcat.id
			WHERE
				threadcat.kind = 3
				OR postcat.kind = 3
		);
		`,
	)
	if err != nil {
		return oops.New(err, "failed to delete the posts")
	}
	fmt.Printf("Deleted %v static pages.\n", res.RowsAffected())

	_, err = tx.Exec(ctx,
		`
		DELETE FROM handmade_thread
		WHERE id IN (
			SELECT thread.id
			FROM
				handmade_thread AS thread
				JOIN handmade_category AS cat ON thread.category_id = cat.id
			WHERE
				cat.kind = 3
		);
		`,
	)
	if err != nil {
		return oops.New(err, "failed to delete the threads")
	}

	_, err = tx.Exec(ctx, `DELETE FROM handmade_category WHERE kind = 3`)
	if err != nil {
		return oops.New(err, "failed to delete the categories")
	}

	return nil
}

func (m DeleteStaticPages) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4"
)

func init() {
	registerMigration(FixPostCategoryId{})
}

type FixPostCategoryId struct{}

func (m FixPostCategoryId) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 4, 26, 0, 0, 0, 1, time.UTC))
}

func (m FixPostCategoryId) Name() string {
	return "FixPostCategoryId"
}

func (m FixPostCategoryId) Description() string {
	return "Copy category id from thread"
}

func (m FixPostCategoryId) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		UPDATE handmade_post
		SET (category_id) = (
			SELECT thread.category_id
			FROM
				handmade_thread AS thread
				JOIN handmade_post AS post ON post.thread_id = thread.id
			WHERE
				post.id = handmade_post.id
		)
		`,
	)
	if err != nil {
		return oops.New(err, "failed to migrate data from categories")
	}

	return nil
}

func (m FixPostCategoryId) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

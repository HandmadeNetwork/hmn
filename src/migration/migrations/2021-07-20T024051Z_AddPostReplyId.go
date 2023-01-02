package migrations

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(AddPostReplyId{})
}

type AddPostReplyId struct{}

func (m AddPostReplyId) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 7, 20, 2, 40, 51, 0, time.UTC))
}

func (m AddPostReplyId) Name() string {
	return "AddPostReplyId"
}

func (m AddPostReplyId) Description() string {
	return "Add a reply id to posts"
}

func (m AddPostReplyId) Up(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx,
		`
		ALTER TABLE handmade_post
			ADD reply_id INT REFERENCES handmade_post (id) ON DELETE SET NULL;
		`,
	)
	if err != nil {
		return oops.New(err, "failed to add columns")
	}

	return nil
}

func (m AddPostReplyId) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

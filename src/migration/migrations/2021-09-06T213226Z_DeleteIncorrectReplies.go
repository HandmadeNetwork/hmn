package migrations

import (
	"context"
	"fmt"
	"time"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v5"
)

func init() {
	registerMigration(DeleteIncorrectReplies{})
}

type DeleteIncorrectReplies struct{}

func (m DeleteIncorrectReplies) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 9, 6, 21, 32, 26, 0, time.UTC))
}

func (m DeleteIncorrectReplies) Name() string {
	return "DeleteIncorrectReplies"
}

func (m DeleteIncorrectReplies) Description() string {
	return "Remove reply ID from posts that are replying to the OP"
}

func (m DeleteIncorrectReplies) Up(ctx context.Context, tx pgx.Tx) error {
	tag, err := tx.Exec(ctx, `
		UPDATE handmade_post
		SET reply_id = NULL
		WHERE id IN (
			SELECT post.id
			FROM
				handmade_post AS post
				JOIN handmade_thread AS thread ON post.thread_id = thread.id
			WHERE
				post.reply_id = thread.first_id
		)
	`)
	if err != nil {
		return oops.New(err, "failed to delete bad reply IDs")
	}

	fmt.Printf("Cleared the reply id on %d posts.\n", tag.RowsAffected())

	return nil
}

func (m DeleteIncorrectReplies) Down(ctx context.Context, tx pgx.Tx) error {
	fmt.Println("This migration was just a fixup and we don't really need a reverse.")
	return nil
}

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
	registerMigration(DeleteBadUnreadInfo{})
}

type DeleteBadUnreadInfo struct{}

func (m DeleteBadUnreadInfo) Version() types.MigrationVersion {
	return types.MigrationVersion(time.Date(2021, 5, 4, 0, 12, 58, 0, time.UTC))
}

func (m DeleteBadUnreadInfo) Name() string {
	return "DeleteBadUnreadInfo"
}

func (m DeleteBadUnreadInfo) Description() string {
	return "Delete invalid tlri and clri"
}

func (m DeleteBadUnreadInfo) Up(ctx context.Context, tx pgx.Tx) error {
	threadNullsResult, err := tx.Exec(ctx, `
		DELETE FROM handmade_threadlastreadinfo
		WHERE
			lastread IS NULL
			OR thread_id IS NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to delete thread entries with null fields")
	}
	fmt.Printf("Deleted %d thread entries with null fields\n", threadNullsResult.RowsAffected())

	catNullsResult, err := tx.Exec(ctx, `
		DELETE FROM handmade_categorylastreadinfo
		WHERE
			lastread IS NULL
			OR category_id IS NULL;
	`)
	if err != nil {
		return oops.New(err, "failed to delete category entries with null fields")
	}
	fmt.Printf("Deleted %d category entries with null fields\n", catNullsResult.RowsAffected())

	return nil
}

func (m DeleteBadUnreadInfo) Down(ctx context.Context, tx pgx.Tx) error {
	panic("Implement me")
}

package migrations

import (
	"context"
	"fmt"

	"git.handmade.network/hmn/hmn/src/migration/types"
	"github.com/jackc/pgx/v4"
)

var All map[types.MigrationVersion]types.Migration = make(map[types.MigrationVersion]types.Migration)

func registerMigration(m types.Migration) {
	All[m.Version()] = m
}

func debugQuery(ctx context.Context, tx pgx.Tx, sql string) {
	rows, err := tx.Query(ctx, sql)
	if err != nil {
		panic(err)
	}
	for rows.Next() {
		vals, _ := rows.Values()
		fmt.Println(vals)
	}
}

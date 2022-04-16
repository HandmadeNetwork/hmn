package hmndata

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
)

type TagQuery struct {
	IDs  []int
	Text []string

	Limit, Offset int
}

func FetchTags(ctx context.Context, dbConn db.ConnOrTx, q TagQuery) ([]*models.Tag, error) {
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch tags")
	defer perf.EndBlock()

	var qb db.QueryBuilder
	qb.Add(
		`
		SELECT $columns
		FROM tags
		WHERE
			TRUE
		`,
	)
	if len(q.IDs) > 0 {
		qb.Add(`AND id = ANY ($?)`, q.IDs)
	}
	if len(q.Text) > 0 {
		qb.Add(`AND text = ANY ($?)`, q.Text)
	}
	if q.Limit > 0 {
		qb.Add(`LIMIT $? OFFSET $?`, q.Limit, q.Offset)
	}

	tags, err := db.Query[models.Tag](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch tags")
	}
	return tags, nil
}

func FetchTag(ctx context.Context, dbConn db.ConnOrTx, q TagQuery) (*models.Tag, error) {
	tags, err := FetchTags(ctx, dbConn, q)
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, db.NotFound
	}
	return tags[0], nil
}

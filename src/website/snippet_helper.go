package website

import (
	"context"

	"git.handmade.network/hmn/hmn/src/oops"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
)

type SnippetQuery struct {
	IDs      []int
	OwnerIDs []int
	Tags     []int

	Limit, Offset int // if empty, no pagination
}

type SnippetAndStuff struct {
	Snippet        models.Snippet
	Owner          *models.User
	Asset          *models.Asset          `db:"asset"`
	DiscordMessage *models.DiscordMessage `db:"discord_message"`
	Tags           []*models.Tag
}

func FetchSnippets(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q SnippetQuery,
) ([]SnippetAndStuff, error) {
	perf := ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch snippets")
	defer perf.EndBlock()

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	var qb db.QueryBuilder
	qb.Add(
		`
		SELECT $columns
		FROM
			handmade_snippet AS snippet
			LEFT JOIN auth_user AS owner ON snippet.owner_id = owner.id
			LEFT JOIN handmade_asset AS asset ON snippet.asset_id = asset.id
			LEFT JOIN handmade_discordmessage AS discord_message ON snippet.discord_message_id = discord_message.id
			LEFT JOIN snippet_tags ON snippet.id = snippet_tags.snippet_id
			LEFT JOIN tags ON snippet_tags.tag_id = tags.id
		WHERE
			TRUE
		`,
	)
	if len(q.IDs) > 0 {
		qb.Add(`AND snippet.id = ANY ($?)`, q.IDs)
	}
	if len(q.OwnerIDs) > 0 {
		qb.Add(`AND snippet.owner_id = ANY ($?)`, q.OwnerIDs)
	}
	if len(q.Tags) > 0 {
		qb.Add(`AND snippet_tags.tag_id = ANY ($?)`, q.Tags)
	}
	if currentUser == nil {
		qb.Add(
			`AND owner.status = $? -- snippet owner is Approved`,
			models.UserStatusApproved,
		)
	} else if !currentUser.IsStaff {
		qb.Add(
			`
			AND (
				owner.status = $? -- snippet owner is Approved
				OR owner.id = $? -- current user is the snippet owner
			)
			`,
			models.UserStatusApproved,
			currentUser.ID,
		)
	}
	qb.Add(`ORDER BY snippet.when DESC, snippet.id ASC`)
	if q.Limit > 0 {
		qb.Add(`LIMIT $? OFFSET $?`, q.Limit, q.Offset)
	}

	type resultRow struct {
		Snippet        models.Snippet         `db:"snippet"`
		Owner          *models.User           `db:"owner"`
		Asset          *models.Asset          `db:"asset"`
		DiscordMessage *models.DiscordMessage `db:"discord_message"`
		Tag            *models.Tag            `db:"tags"`
	}

	it, err := db.Query(ctx, dbConn, resultRow{}, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch threads")
	}
	iresults := it.ToSlice()

	result := make([]SnippetAndStuff, 0, len(iresults)) // allocate extra space because why not
	currentSnippetId := -1
	for _, iresult := range iresults {
		row := *iresult.(*resultRow)

		if row.Snippet.ID != currentSnippetId {
			// we have moved onto a new snippet; make a new entry
			result = append(result, SnippetAndStuff{
				Snippet:        row.Snippet,
				Owner:          row.Owner,
				Asset:          row.Asset,
				DiscordMessage: row.DiscordMessage,
				// no tags! tags next
			})
		}

		if row.Tag != nil {
			result[len(result)-1].Tags = append(result[len(result)-1].Tags, row.Tag)
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to commit transaction")
	}

	return result, nil
}

func FetchSnippet(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	snippetID int,
	q SnippetQuery,
) (SnippetAndStuff, error) {
	q.IDs = []int{snippetID}
	q.Limit = 1
	q.Offset = 0

	res, err := FetchSnippets(ctx, dbConn, currentUser, q)
	if err != nil {
		return SnippetAndStuff{}, oops.New(err, "failed to fetch snippet")
	}

	if len(res) == 0 {
		return SnippetAndStuff{}, db.NotFound
	}

	return res[0], nil
}

func FetchTags(ctx context.Context, dbConn db.ConnOrTx, text []string) ([]*models.Tag, error) {
	perf := ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch snippets")
	defer perf.EndBlock()

	it, err := db.Query(ctx, dbConn, models.Tag{},
		`
		SELECT $columns
		FROM tags
		WHERE text = ANY ($1)
		`,
		text,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch tags")
	}
	itags := it.ToSlice()

	res := make([]*models.Tag, len(itags))
	for i, itag := range itags {
		tag := itag.(*models.Tag)
		res[i] = tag
	}

	return res, nil
}

func FetchTag(ctx context.Context, dbConn db.ConnOrTx, text string) (*models.Tag, error) {
	tags, err := FetchTags(ctx, dbConn, []string{text})
	if err != nil {
		return nil, err
	}
	if len(tags) == 0 {
		return nil, db.NotFound
	}
	return tags[0], nil
}

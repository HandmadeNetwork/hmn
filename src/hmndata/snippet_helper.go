package hmndata

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
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
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch snippets")
	defer perf.EndBlock()

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	if len(q.Tags) > 0 {
		// Get snippet IDs with this tag, then use that in the main query
		type snippetIDRow struct {
			SnippetID int `db:"snippet_id"`
		}
		itSnippetIDs, err := db.Query(ctx, tx, snippetIDRow{},
			`
			SELECT DISTINCT snippet_id
			FROM
				snippet_tags
				JOIN tags ON snippet_tags.tag_id = tags.id
			WHERE
				tags.id = ANY ($1)
			`,
			q.Tags,
		)
		if err != nil {
			return nil, oops.New(err, "failed to get snippet IDs for tag")
		}
		iSnippetIDs := itSnippetIDs.ToSlice()

		q.IDs = make([]int, len(iSnippetIDs))
		for i := range iSnippetIDs {
			q.IDs[i] = iSnippetIDs[i].(*snippetIDRow).SnippetID
		}
	}

	var qb db.QueryBuilder
	qb.Add(
		`
		SELECT $columns
		FROM
			handmade_snippet AS snippet
			LEFT JOIN auth_user AS owner ON snippet.owner_id = owner.id
			LEFT JOIN handmade_asset AS asset ON snippet.asset_id = asset.id
			LEFT JOIN handmade_discordmessage AS discord_message ON snippet.discord_message_id = discord_message.id
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
	}

	it, err := db.Query(ctx, tx, resultRow{}, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch threads")
	}
	iresults := it.ToSlice()

	result := make([]SnippetAndStuff, len(iresults)) // allocate extra space because why not
	snippetIDs := make([]int, len(iresults))
	for i, iresult := range iresults {
		row := *iresult.(*resultRow)

		result[i] = SnippetAndStuff{
			Snippet:        row.Snippet,
			Owner:          row.Owner,
			Asset:          row.Asset,
			DiscordMessage: row.DiscordMessage,
			// no tags! tags next
		}
		snippetIDs[i] = row.Snippet.ID
	}

	// Fetch tags
	type snippetTagRow struct {
		SnippetID int         `db:"snippet_tags.snippet_id"`
		Tag       *models.Tag `db:"tags"`
	}
	itSnippetTags, err := db.Query(ctx, tx, snippetTagRow{},
		`
		SELECT $columns
		FROM
			snippet_tags
			JOIN tags ON snippet_tags.tag_id = tags.id
		WHERE
			snippet_tags.snippet_id = ANY($1)
		`,
		snippetIDs,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch tags for snippets")
	}
	iSnippetTags := itSnippetTags.ToSlice()

	// associate tags with snippets
	resultBySnippetId := make(map[int]*SnippetAndStuff)
	for i := range result {
		resultBySnippetId[result[i].Snippet.ID] = &result[i]
	}
	for _, iSnippetTag := range iSnippetTags {
		snippetTag := iSnippetTag.(*snippetTagRow)
		item := resultBySnippetId[snippetTag.SnippetID]
		item.Tags = append(item.Tags, snippetTag.Tag)
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

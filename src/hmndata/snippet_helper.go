package hmndata

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
)

type SnippetQuery struct {
	IDs               []int
	OwnerIDs          []int
	ProjectIDs        []int
	Tags              []int
	DiscordMessageIDs []string

	Limit, Offset int // if empty, no pagination
}

type SnippetAndStuff struct {
	Snippet        models.Snippet
	Owner          *models.User
	Asset          *models.Asset          `db:"asset"`
	DiscordMessage *models.DiscordMessage `db:"discord_message"`
	Tags           []*models.Tag
	Projects       []*ProjectAndStuff
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

	var tagSnippetIDs []int
	if len(q.Tags) > 0 {
		// Get snippet IDs with this tag, then use that in the main query
		snippetIDs, err := db.QueryScalar[int](ctx, tx,
			`
			SELECT DISTINCT snippet_id
			FROM
				snippet_tag
				JOIN tag ON snippet_tag.tag_id = tag.id
			WHERE
				tag.id = ANY ($1)
			`,
			q.Tags,
		)
		if err != nil {
			return nil, oops.New(err, "failed to get snippet IDs for tag")
		}

		tagSnippetIDs = snippetIDs
	}

	var projectSnippetIDs []int
	if len(q.ProjectIDs) > 0 {
		// Get snippet IDs for these projects, then use that in the main query
		snippetIDs, err := db.QueryScalar[int](ctx, tx,
			`
			SELECT DISTINCT snippet_id
			FROM
				snippet_project
			WHERE
				project_id = ANY ($1)
			`,
			q.ProjectIDs,
		)
		if err != nil {
			return nil, oops.New(err, "failed to get snippet IDs for tag")
		}

		projectSnippetIDs = snippetIDs
	}

	var qb db.QueryBuilder
	qb.Add(
		`
		SELECT $columns
		FROM
			snippet
			LEFT JOIN hmn_user AS owner ON snippet.owner_id = owner.id
			LEFT JOIN asset AS avatar ON avatar.id = owner.avatar_asset_id
			LEFT JOIN asset ON snippet.asset_id = asset.id
			LEFT JOIN discord_message ON snippet.discord_message_id = discord_message.id
		WHERE
			TRUE
		`,
	)
	allSnippetIDs := make([]int, 0, len(q.IDs)+len(tagSnippetIDs)+len(projectSnippetIDs))
	allSnippetIDs = append(allSnippetIDs, q.IDs...)
	allSnippetIDs = append(allSnippetIDs, tagSnippetIDs...)
	allSnippetIDs = append(allSnippetIDs, projectSnippetIDs...)
	if len(allSnippetIDs) == 0 {
		// We already managed to filter out all snippets, and all further
		// parts of this query are more filters, so we can just fail everything
		// else from right here.
		qb.Add(`AND FALSE`)
	} else if len(q.OwnerIDs) > 0 {
		qb.Add(`AND (snippet.id = ANY ($?) OR snippet.owner_id = ANY ($?))`, allSnippetIDs, q.OwnerIDs)
	} else {
		qb.Add(`AND snippet.id = ANY ($?)`, allSnippetIDs)
		if len(q.OwnerIDs) > 0 {
			qb.Add(`AND snippet.owner_id = ANY ($?)`, q.OwnerIDs)
		}
	}
	if len(q.DiscordMessageIDs) > 0 {
		qb.Add(`AND snippet.discord_message_id = ANY ($?)`, q.DiscordMessageIDs)
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
		AvatarAsset    *models.Asset          `db:"avatar"`
		Asset          *models.Asset          `db:"asset"`
		DiscordMessage *models.DiscordMessage `db:"discord_message"`
	}

	results, err := db.Query[resultRow](ctx, tx, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch threads")
	}

	result := make([]SnippetAndStuff, len(results)) // allocate extra space because why not
	snippetIDs := make([]int, len(results))
	for i, row := range results {
		if results[i].Owner != nil {
			results[i].Owner.AvatarAsset = results[i].AvatarAsset
		}
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
		SnippetID int         `db:"snippet_tag.snippet_id"`
		Tag       *models.Tag `db:"tag"`
	}
	snippetTags, err := db.Query[snippetTagRow](ctx, tx,
		`
		SELECT $columns
		FROM
			snippet_tag
			JOIN tag ON snippet_tag.tag_id = tag.id
		WHERE
			snippet_tag.snippet_id = ANY($1)
		`,
		snippetIDs,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch tags for snippets")
	}

	// associate tags with snippets
	resultBySnippetId := make(map[int]*SnippetAndStuff)
	for i := range result {
		resultBySnippetId[result[i].Snippet.ID] = &result[i]
	}
	for _, snippetTag := range snippetTags {
		item := resultBySnippetId[snippetTag.SnippetID]
		item.Tags = append(item.Tags, snippetTag.Tag)
	}

	// Fetch projects
	type snippetProjectRow struct {
		SnippetID int `db:"snippet_id"`
		ProjectID int `db:"project_id"`
	}
	snippetProjects, err := db.Query[snippetProjectRow](ctx, tx,
		`
		SELECT $columns
		FROM snippet_project
		WHERE snippet_id = ANY($1)
		`,
		snippetIDs,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch project ids for snippets")
	}
	var projectIds []int
	for _, sp := range snippetProjects {
		projectIds = append(projectIds, sp.ProjectID)
	}
	projects, err := FetchProjects(ctx, tx, currentUser, ProjectsQuery{ProjectIDs: projectIds})
	if err != nil {
		return nil, oops.New(err, "failed to fetch projects for snippets")
	}
	projectMap := make(map[int]*ProjectAndStuff)
	for i := range projects {
		projectMap[projects[i].Project.ID] = &projects[i]
	}
	for _, sp := range snippetProjects {
		snip, hasResult := resultBySnippetId[sp.SnippetID]
		proj, hasProj := projectMap[sp.ProjectID]
		if hasResult && hasProj {
			snip.Projects = append(snip.Projects, proj)
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

func FetchSnippetForDiscordMessage(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	discordMessageID string,
	q SnippetQuery,
) (SnippetAndStuff, error) {
	q.DiscordMessageIDs = []string{discordMessageID}
	q.Limit = 1
	q.Offset = 0

	res, err := FetchSnippets(ctx, dbConn, currentUser, q)
	if err != nil {
		return SnippetAndStuff{}, oops.New(err, "failed to fetch snippet for Discord message")
	}

	if len(res) == 0 {
		return SnippetAndStuff{}, db.NotFound
	}

	return res[0], nil
}

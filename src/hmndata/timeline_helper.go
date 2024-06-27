package hmndata

import (
	"context"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
)

/*
WITH snippet_item AS (
	SELECT id,
    "when",
    'snippet' AS timeline_type,
    owner_id,
    '' AS title,
    _description_html AS parsed_desc,
    description AS raw_desc,
    asset_id,
    discord_message_id,
    url,
    0 AS project_id,
    0 AS thread_id,
    0 AS subforum_id,
    0 AS thread_type,
    false AS stream_ended,
    NOW() AS stream_end_time,
    '' AS twitch_login,
    '' AS stream_id
	FROM snippet
),
post_item AS (
	SELECT post.id,
    postdate AS "when",
    'post' AS timeline_type,
    author_id AS owner_id,
    thread.title AS title,
    post_version.text_parsed AS parsed_desc,
    post_version.text_raw AS raw_desc,
    NULL::uuid AS asset_id,
    NULL AS discord_message_id,
    NULL AS url,
    post.project_id,
    thread_id,
    subforum_id,
    0 AS thread_type,
    false AS stream_ended,
    NOW() AS stream_end_time,
    '' AS twitch_login,
    '' AS stream_id
	FROM post
	JOIN thread ON thread.id = post.thread_id
	JOIN post_version ON post_version.id = post.current_id
	WHERE post.deleted = false AND thread.deleted = false
)
SELECT * from snippet_item
UNION ALL
SELECT * from post_item
ORDER BY "when" DESC LIMIT 100;
*/

type TimelineQuery struct {
	OwnerIDs   []int
	ProjectIDs []int

	SkipSnippets bool
	SkipPosts    bool

	Limit, Offset int
}

type TimelineItemAndStuff struct {
	Item           models.TimelineItem    `db:"item"`
	Owner          *models.User           `db:"owner"`
	AvatarAsset    *models.Asset          `db:"avatar"`
	Asset          *models.Asset          `db:"asset"`
	DiscordMessage *models.DiscordMessage `db:"discord_message"`
	Projects       []*ProjectAndStuff
}

func FetchTimeline(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q TimelineQuery,
) ([]*TimelineItemAndStuff, error) {
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch timeline")
	defer perf.EndBlock()

	var qb db.QueryBuilder

	itemSelects := []string{}
	if !q.SkipSnippets {
		itemSelects = append(itemSelects, "SELECT * from snippet_item")
	}
	if !q.SkipPosts {
		itemSelects = append(itemSelects, "SELECT * from post_item")
	}

	if len(itemSelects) == 0 {
		return nil, nil
	}

	itemSelect := strings.Join(itemSelects, " UNION ALL ")

	qb.Add(
		`
		WITH snippet_item AS (
			SELECT id,
			"when",
			'snippet' AS timeline_type,
			owner_id,
			'' AS title,
			_description_html AS parsed_desc,
			description AS raw_desc,
			asset_id,
			discord_message_id,
			url,
			0 AS project_id,
			0 AS thread_id,
			0 AS subforum_id,
			0 AS thread_type,
			TRUE AS first_post
			FROM snippet
			WHERE TRUE
		`,
	)
	if len(q.ProjectIDs) > 0 {
		qb.Add(
			`
			AND (
				SELECT count(*)
				FROM snippet_project
				WHERE
					snippet_project.snippet_id = snippet.id
					AND
					snippet_project.project_id = ANY($?)
				) > 0
			`,
			q.ProjectIDs,
		)
	}
	qb.Add(
		`
		),
		post_item AS (
			SELECT post.id,
			postdate AS "when",
			'post' AS timeline_type,
			author_id AS owner_id,
			thread.title AS title,
			post_version.text_parsed AS parsed_desc,
			post_version.text_raw AS raw_desc,
			NULL::uuid AS asset_id,
			NULL AS discord_message_id,
			NULL AS url,
			post.project_id,
			thread_id,
			subforum_id,
			thread_type,
			(post.id = thread.first_id) AS first_post
			FROM post
			JOIN thread ON thread.id = thread_id
			JOIN post_version ON post_version.id = current_id
			WHERE post.deleted = false AND thread.deleted = false
		`,
	)
	if len(q.ProjectIDs) > 0 {
		qb.Add(`AND post.project_id = ANY($?)`, q.ProjectIDs)
	}
	qb.Add(
		`
		),
		item AS (
		`,
	)
	qb.Add(itemSelect)
	qb.Add(
		`
		)
		SELECT $columns FROM item
		LEFT JOIN thread ON thread.id = thread_id
		LEFT JOIN hmn_user AS owner ON owner_id = owner.id
		LEFT JOIN asset AS avatar ON avatar.id = owner.avatar_asset_id
		LEFT JOIN asset ON asset_id = asset.id
		LEFT JOIN discord_message ON discord_message_id = discord_message.id
		WHERE TRUE
		`,
	)

	if len(q.OwnerIDs) > 0 {
		qb.Add(`AND owner_id = ANY($?)`, q.OwnerIDs)
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

	qb.Add(
		`
		ORDER BY "when" DESC
		`,
	)
	if q.Limit > 0 {
		qb.Add(`LIMIT $? OFFSET $?`, q.Limit, q.Offset)
	}

	results, err := db.Query[TimelineItemAndStuff](ctx, dbConn, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch timeline items")
	}

	for idx := range results {
		if results[idx].Owner != nil {
			results[idx].Owner.AvatarAsset = results[idx].AvatarAsset
		}
	}

	var projectIds []int
	var snippetIds []int
	projectTargets := make(map[int][]*TimelineItemAndStuff)
	snippetItems := make(map[int]*TimelineItemAndStuff)
	for _, r := range results {
		if r.Item.ProjectID != 0 {
			projectIds = append(projectIds, r.Item.ProjectID)
			projectTargets[r.Item.ProjectID] = append(projectTargets[r.Item.ProjectID], r)
		}
		if r.Item.Type == models.TimelineItemTypeSnippet {
			snippetIds = append(snippetIds, r.Item.ID)
			snippetItems[r.Item.ID] = r
		}
	}

	type snippetProjectRow struct {
		SnippetID int `db:"snippet_id"`
		ProjectID int `db:"project_id"`
	}
	snippetProjects, err := db.Query[snippetProjectRow](ctx, dbConn,
		`
		SELECT $columns
		FROM snippet_project
		WHERE snippet_id = ANY($1)
		`,
		snippetIds,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch project ids for timeline")
	}

	for _, sp := range snippetProjects {
		projectIds = append(projectIds, sp.ProjectID)
		projectTargets[sp.ProjectID] = append(projectTargets[sp.ProjectID], snippetItems[sp.SnippetID])
	}

	projects, err := FetchProjects(ctx, dbConn, currentUser, ProjectsQuery{
		ProjectIDs:    projectIds,
		IncludeHidden: true,
		Lifecycles:    models.AllProjectLifecycles,
	})
	if err != nil {
		return nil, oops.New(err, "failed to fetch projects for timeline")
	}
	for pIdx := range projects {
		targets := projectTargets[projects[pIdx].Project.ID]
		for _, t := range targets {
			t.Projects = append(t.Projects, &projects[pIdx])
		}
	}

	return results, nil
}

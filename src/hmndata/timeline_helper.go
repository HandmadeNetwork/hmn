package hmndata

import (
	"context"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
)

type TimelineQuery struct {
	OwnerIDs   []int
	ProjectIDs []int

	SkipSnippets bool
	SkipPosts    bool

	IncludePostDescription bool

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
	defer perf.StartBlock(ctx, "TIMELINE", "Fetch timeline").End()

	var qb db.QueryBuilder
	qb.AddName("Fetch base timeline data")

	currentUserId := -1
	if currentUser != nil {
		currentUserId = currentUser.ID
	}

	currentUserIsAdmin := false
	if currentUser != nil {
		currentUserIsAdmin = currentUser.IsStaff
	}

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
		WITH visible_project AS (
			SELECT *
			FROM project
			WHERE
				$? = true
				OR project.id = $?
				OR (SELECT count(*) > 0 FROM user_project WHERE project_id = project.id AND user_id = $?)
				OR (
					project.lifecycle = ANY($?)
					AND NOT project.hidden
					AND (SELECT every(hmn_user.status = $?)
						 FROM user_project
						 JOIN hmn_user ON hmn_user.id = user_project.user_id
						 WHERE user_project.project_id = project.id
					)
				)
		),`,
		currentUserIsAdmin,
		models.HMNProjectID,
		currentUserId,
		models.VisibleProjectLifecycles,
		models.UserStatusApproved,
	)
	qb.Add(
		`
		snippet_item AS (
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
	if len(q.ProjectIDs)+len(q.OwnerIDs) > 0 {
		qb.Add(`AND (`)
		if len(q.ProjectIDs) > 0 {
			qb.Add(
				`
				(
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
		} else {
			qb.Add("FALSE")
		}
		qb.Add(" OR ")
		if len(q.OwnerIDs) > 0 {
			qb.Add(`owner_id = ANY($?)`, q.OwnerIDs)
		} else {
			qb.Add("FALSE")
		}
		qb.Add(`)`)
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
		`,
	)
	if q.IncludePostDescription {
		qb.Add(
			`
			post_version.text_parsed AS parsed_desc,
			post_version.text_raw AS raw_desc,
			`,
		)
	} else {
		qb.Add(
			`
			'' AS parsed_desc,
			'' AS raw_desc,
			`,
		)
	}
	qb.Add(
		`
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
			JOIN visible_project ON visible_project.id = post.project_id
			WHERE
				post.deleted = false AND thread.deleted = false
		`,
	)
	if len(q.OwnerIDs)+len(q.ProjectIDs) > 0 {
		qb.Add(`AND (`)
		if len(q.ProjectIDs) > 0 {
			qb.Add(`post.project_id = ANY($?)`, q.ProjectIDs)
		} else {
			qb.Add("FALSE")
		}
		qb.Add(" OR ")
		if len(q.OwnerIDs) > 0 {
			qb.Add(`post.author_id = ANY($?)`, q.OwnerIDs)
		} else {
			qb.Add("FALSE")
		}
		qb.Add(`)`)
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

	projectsSeen := make(map[int]bool)
	var projectIds []int
	var snippetIds []int
	projectTargets := make(map[int][]*TimelineItemAndStuff)
	snippetItems := make(map[int]*TimelineItemAndStuff)
	for _, r := range results {
		if r.Item.ProjectID != 0 {
			if _, found := projectsSeen[r.Item.ProjectID]; !found {
				projectIds = append(projectIds, r.Item.ProjectID)
				projectsSeen[r.Item.ProjectID] = true
			}

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
		---- Fetch snippet projects
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
		if _, found := projectsSeen[sp.ProjectID]; !found {
			projectIds = append(projectIds, sp.ProjectID)
			projectsSeen[sp.ProjectID] = true
		}
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

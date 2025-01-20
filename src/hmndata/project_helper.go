package hmndata

import (
	"context"
	"fmt"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
)

type ProjectTypeQuery int

const (
	PersonalProjects ProjectTypeQuery = 1 << iota
	OfficialProjects
)

type ProjectsQuery struct {
	// Available on all project queries. By default, you will get projects that
	// are generally visible to all users.
	Lifecycles    []models.ProjectLifecycle // If empty, defaults to visible lifecycles. Do not conflate this with permissions; those are checked separately.
	Types         ProjectTypeQuery          // bitfield
	FeaturedOnly  bool
	IncludeHidden bool

	// Ignored when using FetchProject
	ProjectIDs []int    // if empty, all projects
	Slugs      []string // if empty, all projects
	OwnerIDs   []int    // if empty, all projects
	JamSlugs   []string // if empty, all projects

	// Ignored when using CountProjects
	Limit, Offset int // if empty, no pagination
	OrderBy       string
}

type ProjectAndStuff struct {
	Project        models.Project
	LogoLightAsset *models.Asset
	LogoDarkAsset  *models.Asset
	HeaderImage    *models.Asset
	Owners         []*models.User
	Tag            *models.Tag
}

func (p *ProjectAndStuff) TagText() string {
	if p.Tag == nil {
		return ""
	} else {
		return p.Tag.Text
	}
}

func FetchProjects(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q ProjectsQuery,
) ([]ProjectAndStuff, error) {
	defer perf.StartBlock(ctx, "PROJECT", "Fetch projects").End()

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	type projectRow struct {
		Project        models.Project `db:"project"`
		LogoLightAsset *models.Asset  `db:"logolight_asset"`
		LogoDarkAsset  *models.Asset  `db:"logodark_asset"`
		HeaderAsset    *models.Asset  `db:"header_asset"`
		Tag            *models.Tag    `db:"tag"`
	}

	// Fetch all valid projects (not yet subject to user permission checks)
	var qb db.QueryBuilder
	qb.AddName("Fetch projects")
	if len(q.OrderBy) > 0 {
		qb.Add(`SELECT * FROM (`)
	}
	qb.Add(`
		SELECT DISTINCT ON (project.id) $columns
		FROM
			project
			LEFT JOIN asset AS logolight_asset ON logolight_asset.id = project.logolight_asset_id
			LEFT JOIN asset AS logodark_asset ON logodark_asset.id = project.logodark_asset_id
			LEFT JOIN asset AS header_asset ON header_asset.id = project.header_asset_id
			LEFT JOIN tag ON project.tag = tag.id
	`)
	if len(q.OwnerIDs) > 0 {
		qb.Add(
			`
			JOIN (
				SELECT project_id, array_agg(user_id) AS owner_ids
				FROM user_project
				WHERE user_id = ANY ($?)
				GROUP BY project_id
			) AS owner_filter ON project.id = owner_filter.project_id
			`,
			q.OwnerIDs,
		)
	}

	if len(q.JamSlugs) > 0 {
		qb.Add(
			`
			JOIN jam_project ON jam_project.project_id = project.id
			`,
		)
	}

	// Filters (permissions are checked after the query, in Go)
	qb.Add(`
		WHERE
			TRUE
	`)
	if len(q.Lifecycles) > 0 {
		qb.Add(`AND project.lifecycle = ANY ($?)`, q.Lifecycles)
	} else {
		qb.Add(`AND project.lifecycle = ANY ($?)`, models.VisibleProjectLifecycles)
	}
	if q.Types != 0 {
		qb.Add(`AND (FALSE`)
		if q.Types&PersonalProjects != 0 {
			qb.Add(`OR project.personal`)
		}
		if q.Types&OfficialProjects != 0 {
			qb.Add(`OR NOT project.personal`)
		}
		qb.Add(`)`)
	}
	if q.FeaturedOnly {
		qb.Add(`AND project.featured`)
	}
	if !q.IncludeHidden {
		qb.Add(`AND NOT project.hidden`)
	}
	if len(q.ProjectIDs) > 0 {
		qb.Add(`AND project.id = ANY ($?)`, q.ProjectIDs)
	}
	if len(q.Slugs) > 0 {
		qb.Add(`AND (project.slug != '' AND (project.slug = ANY ($?) OR $? && project.slug_aliases))`, q.Slugs, q.Slugs)
	}
	if len(q.JamSlugs) > 0 {
		qb.Add(`AND (jam_project.jam_slug = ANY ($?) AND jam_project.participating = TRUE)`, q.JamSlugs)
	}

	// Output
	if q.Limit > 0 {
		qb.Add(`LIMIT $? OFFSET $?`, q.Limit, q.Offset)
	}
	if len(q.OrderBy) > 0 {
		qb.Add(fmt.Sprintf(`) q ORDER BY %s`, q.OrderBy))
	}

	// Do the query
	projectRows, err := db.Query[projectRow](ctx, tx, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch projects")
	}

	// Fetch project owners to do permission checks
	projectIds := make([]int, len(projectRows))
	for i, p := range projectRows {
		projectIds[i] = p.Project.ID
	}
	projectOwners, err := FetchMultipleProjectsOwners(ctx, tx, projectIds)
	if err != nil {
		return nil, err
	}

	b := perf.StartBlock(ctx, "PROJECT", "Compute permissions")
	var res []ProjectAndStuff
	for i, p := range projectRows {
		owners := projectOwners[i].Owners

		/*
			Here's the rundown on project permissions:

			- In general, users can only see projects that are Generally Visible.
			- As an exception, users can always see projects that they own.
			- As an exception, staff can always see every project.

			A project is Generally Visible if all the following conditions are true:
			- The project has a "visible" lifecycle (per models.VisibleProjectLifecycles)
			- The project is not hidden
			- One of the following is true:
				- The project is official
				- The project is personal and all of the project's owners are approved

			As an exception, the HMN project is always generally visible.

			See https://www.notion.so/handmade-network/Technical-Plan-a11aaa9ea2f14d9a95f7d7780edd789c
		*/

		currentUserIsOwner := false
		allOwnersApproved := true
		for _, owner := range owners {
			if owner.Status != models.UserStatusApproved {
				allOwnersApproved = false
			}
			if currentUser != nil && owner.ID == currentUser.ID {
				currentUserIsOwner = true
			}
		}

		projectGenerallyVisible := true &&
			p.Project.Lifecycle.In(models.VisibleProjectLifecycles) &&
			!p.Project.Hidden &&
			(!p.Project.Personal || allOwnersApproved || p.Project.IsHMN())
		if p.Project.IsHMN() {
			projectGenerallyVisible = true // hard override
		}

		projectVisible := false ||
			projectGenerallyVisible ||
			currentUserIsOwner ||
			(currentUser != nil && currentUser.IsStaff)

		if projectVisible {
			res = append(res, ProjectAndStuff{
				Project:        p.Project,
				LogoLightAsset: p.LogoLightAsset,
				LogoDarkAsset:  p.LogoDarkAsset,
				HeaderImage:    p.HeaderAsset,
				Owners:         owners,
				Tag:            p.Tag,
			})
		}
	}
	b.End()

	err = tx.Commit(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to commit transaction")
	}

	return res, nil
}

/*
Fetches a single project. A wrapper around FetchProjects.

Returns db.NotFound if no result is found.
*/
func FetchProject(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	projectID int,
	q ProjectsQuery,
) (ProjectAndStuff, error) {
	defer perf.StartBlock(ctx, "PROJECT", "Fetch project").End()

	q.ProjectIDs = []int{projectID}
	q.Limit = 1
	q.Offset = 0

	res, err := FetchProjects(ctx, dbConn, currentUser, q)
	if err != nil {
		return ProjectAndStuff{}, oops.New(err, "failed to fetch project")
	}

	if len(res) == 0 {
		return ProjectAndStuff{}, db.NotFound
	}

	return res[0], nil
}

/*
Fetches a single project by slug. A wrapper around FetchProjects.

Returns db.NotFound if no result is found.
*/
func FetchProjectBySlug(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	projectSlug string,
	q ProjectsQuery,
) (ProjectAndStuff, error) {
	defer perf.StartBlock(ctx, "PROJECT", "Fetch project by slug").End()

	q.Slugs = []string{projectSlug}
	q.Limit = 1
	q.Offset = 0

	res, err := FetchProjects(ctx, dbConn, currentUser, q)
	if err != nil {
		return ProjectAndStuff{}, oops.New(err, "failed to fetch project")
	}

	if len(res) == 0 {
		return ProjectAndStuff{}, db.NotFound
	}

	return res[0], nil
}

func CountProjects(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q ProjectsQuery,
) (int, error) {
	defer perf.StartBlock(ctx, "PROJECT", "Count projects").End()

	q.Limit = 0
	q.Offset = 0

	// I'm lazy and there probably won't ever be that many projects.
	projects, err := FetchProjects(ctx, dbConn, currentUser, q)
	if err != nil {
		return 0, oops.New(err, "failed to fetch projects")
	}

	return len(projects), nil
}

type ProjectOwners struct {
	ProjectID int
	Owners    []*models.User
}

// Fetches all owners for multiple projects. Does NOT check permissions on the
// project IDs, since the assumption is that you will check permissions on the
// projects themselves before using any of this data.
//
// The returned slice will always have one entry for each project ID given, in
// the same order as they were provided. If there are duplicate project IDs in
// projectIds, the results will be wrong, so don't do that.
//
// This function does not verify that the requested projects do in fact exist.
func FetchMultipleProjectsOwners(
	ctx context.Context,
	dbConn db.ConnOrTx,
	projectIds []int,
) ([]ProjectOwners, error) {
	defer perf.StartBlock(ctx, "PROJECT", "Fetch owners for multiple projects").End()

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	// Fetch all user/project pairs for the given projects
	type userProject struct {
		UserID    int `db:"user_id"`
		ProjectID int `db:"project_id"`
	}
	userProjects, err := db.Query[userProject](ctx, tx,
		`
		---- Fetch user/project pairs
		SELECT $columns
		FROM user_project
		WHERE project_id = ANY($1)
		`,
		projectIds,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch project IDs")
	}

	// Get the unique user IDs from this set and fetch the users from the db
	var userIds []int
	for _, userProject := range userProjects {
		addUserId := true
		for _, uid := range userIds {
			if uid == userProject.UserID {
				addUserId = false
			}
		}
		if addUserId {
			userIds = append(userIds, userProject.UserID)
		}
	}
	users, err := FetchUsers(ctx, tx, nil, UsersQuery{
		UserIDs:   userIds,
		AnyStatus: true,
	})
	if err != nil {
		return nil, oops.New(err, "failed to fetch users for projects")
	}

	// Build the final result set with real user data
	res := make([]ProjectOwners, len(projectIds))
	for i, pid := range projectIds {
		res[i] = ProjectOwners{ProjectID: pid}
	}
	for _, userProject := range userProjects {
		// Get a pointer to the existing record in the result
		var projectOwners *ProjectOwners
		for i := range res {
			if res[i].ProjectID == userProject.ProjectID {
				projectOwners = &res[i]
			}
		}

		// Get the full user record we fetched
		var user *models.User
		for _, u := range users {
			if u.ID == userProject.UserID {
				user = u
			}
		}
		if user == nil {
			panic("we apparently failed to fetch a project's owner")
		}

		// Slam 'em together
		projectOwners.Owners = append(projectOwners.Owners, user)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to commit transaction")
	}

	return res, nil
}

// Fetches project owners for a single project. It is subject to all the same
// restrictions as FetchMultipleProjectsOwners.
func FetchProjectOwners(
	ctx context.Context,
	dbConn db.ConnOrTx,
	projectId int,
) ([]*models.User, error) {
	defer perf.StartBlock(ctx, "PROJECT", "Fetch project owners").End()

	projectOwners, err := FetchMultipleProjectsOwners(ctx, dbConn, []int{projectId})
	if err != nil {
		return nil, err
	}

	return projectOwners[0].Owners, nil
}

func UrlContextForProject(p *models.Project) *hmnurl.UrlContext {
	return &hmnurl.UrlContext{
		PersonalProject: p.Personal,
		ProjectID:       p.ID,
		ProjectSlug:     p.Slug,
		ProjectName:     p.Name,
	}
}

func SetProjectTag(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	projectID int,
	tagText string,
) (*models.Tag, error) {
	defer perf.StartBlock(ctx, "PROJECT", "Set project tag").End()

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	p, err := FetchProject(ctx, tx, currentUser, projectID, ProjectsQuery{
		Lifecycles:    models.AllProjectLifecycles,
		IncludeHidden: true,
	})
	if err != nil {
		return nil, oops.New(err, "Failed to fetch project")
	}

	var resultTag *models.Tag
	if tagText == "" {
		// Once a project's tag is set, it cannot be unset. Return the existing tag.
		resultTag = p.Tag
	} else if p.Project.TagID == nil {
		// Create a tag
		tag, err := db.QueryOne[models.Tag](ctx, tx,
			`
			---- Create a new tag
			INSERT INTO tag (text) VALUES ($1)
			RETURNING $columns
			`,
			tagText,
		)
		if err != nil {
			return nil, oops.New(err, "failed to create new tag for project")
		}
		resultTag = tag

		// Attach it to the project
		_, err = tx.Exec(ctx,
			`
			---- Associate tag with project
			UPDATE project
			SET tag = $1
			WHERE id = $2
			`,
			resultTag.ID, projectID,
		)
		if err != nil {
			return nil, oops.New(err, "failed to attach new tag to project")
		}
	} else {
		// Update the text of an existing one
		tag, err := db.QueryOne[models.Tag](ctx, tx,
			`
			---- Update the text of the existing tag
			UPDATE tag
			SET text = $1
			WHERE id = (SELECT tag FROM project WHERE id = $2)
			RETURNING $columns
			`,
			tagText, projectID,
		)
		if err != nil {
			return nil, oops.New(err, "failed to update existing tag")
		}
		resultTag = tag
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to commit transaction")
	}

	return resultTag, nil
}

func UpdateSnippetLastPostedForAllProjects(ctx context.Context, dbConn db.ConnOrTx) error {
	_, err := dbConn.Exec(ctx,
		`
		---- Update snippet_last_posted for everything
		UPDATE project p SET (snippet_last_posted, all_last_updated) = (
			SELECT
				COALESCE(MAX(s."when"), 'epoch'),
				GREATEST(p.forum_last_updated, p.blog_last_updated, p.annotation_last_updated, MAX(s."when"))
			FROM
				snippet s
				JOIN snippet_project sp ON s.id = sp.snippet_id
			WHERE sp.project_id = p.id
		)
		`,
	)
	return err
}

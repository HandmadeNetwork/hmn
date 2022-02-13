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
	IncludeHidden bool

	// Ignored when using FetchProject
	ProjectIDs []int    // if empty, all projects
	Slugs      []string // if empty, all projects
	OwnerIDs   []int    // if empty, all projects

	// Ignored when using CountProjects
	Limit, Offset int // if empty, no pagination
	OrderBy       string
}

type ProjectAndStuff struct {
	Project        models.Project
	LogoLightAsset *models.Asset `db:"logolight_asset"`
	LogoDarkAsset  *models.Asset `db:"logodark_asset"`
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
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch projects")
	defer perf.EndBlock()

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	type projectRow struct {
		Project        models.Project `db:"project"`
		LogoLightAsset *models.Asset  `db:"logolight_asset"`
		LogoDarkAsset  *models.Asset  `db:"logodark_asset"`
		Tag            *models.Tag    `db:"tags"`
	}

	// Fetch all valid projects (not yet subject to user permission checks)
	var qb db.QueryBuilder
	if len(q.OrderBy) > 0 {
		qb.Add(`SELECT * FROM (`)
	}
	qb.Add(`
		SELECT DISTINCT ON (project.id) $columns
		FROM
			handmade_project AS project
			LEFT JOIN handmade_asset AS logolight_asset ON logolight_asset.id = project.logolight_asset_id
			LEFT JOIN handmade_asset AS logodark_asset ON logodark_asset.id = project.logodark_asset_id
			LEFT JOIN tags ON project.tag = tags.id
	`)
	if len(q.OwnerIDs) > 0 {
		qb.Add(
			`
			JOIN (
				SELECT project_id, array_agg(user_id) AS owner_ids
				FROM handmade_user_projects
				WHERE user_id = ANY ($?)
				GROUP BY project_id
			) AS owner_filter ON project.id = owner_filter.project_id
			`,
			q.OwnerIDs,
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
	if !q.IncludeHidden {
		qb.Add(`AND NOT project.hidden`)
	}
	if len(q.ProjectIDs) > 0 {
		qb.Add(`AND project.id = ANY ($?)`, q.ProjectIDs)
	}
	if len(q.Slugs) > 0 {
		qb.Add(`AND (project.slug != '' AND project.slug = ANY ($?))`, q.Slugs)
	}

	// Output
	if q.Limit > 0 {
		qb.Add(`LIMIT $? OFFSET $?`, q.Limit, q.Offset)
	}
	if len(q.OrderBy) > 0 {
		qb.Add(fmt.Sprintf(`) q ORDER BY %s`, q.OrderBy))
	}

	// Do the query
	iprojects, err := db.Query(ctx, dbConn, projectRow{}, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch projects")
	}

	// Fetch project owners to do permission checks
	projectIds := make([]int, len(iprojects))
	for i, iproject := range iprojects {
		projectIds[i] = iproject.(*projectRow).Project.ID
	}
	projectOwners, err := FetchMultipleProjectsOwners(ctx, tx, projectIds)
	if err != nil {
		return nil, err
	}

	var res []ProjectAndStuff
	for i, iproject := range iprojects {
		row := iproject.(*projectRow)
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
			row.Project.Lifecycle.In(models.VisibleProjectLifecycles) &&
			!row.Project.Hidden &&
			(!row.Project.Personal || allOwnersApproved || row.Project.IsHMN())
		if row.Project.IsHMN() {
			projectGenerallyVisible = true // hard override
		}

		projectVisible := false ||
			projectGenerallyVisible ||
			currentUserIsOwner ||
			(currentUser != nil && currentUser.IsStaff)

		if projectVisible {
			res = append(res, ProjectAndStuff{
				Project:        row.Project,
				LogoLightAsset: row.LogoLightAsset,
				LogoDarkAsset:  row.LogoDarkAsset,
				Owners:         owners,
				Tag:            row.Tag,
			})
		}
	}

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
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Count projects")
	defer perf.EndBlock()

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
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch owners for multiple projects")
	defer perf.EndBlock()

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
	iuserprojects, err := db.Query(ctx, tx, userProject{},
		`
		SELECT $columns
		FROM handmade_user_projects
		WHERE project_id = ANY($1)
		`,
		projectIds,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch project IDs")
	}

	// Get the unique user IDs from this set and fetch the users from the db
	var userIds []int
	for _, iuserproject := range iuserprojects {
		userProject := iuserproject.(*userProject)

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
	type userQuery struct {
		User models.User `db:"auth_user"`
	}
	iusers, err := db.Query(ctx, tx, userQuery{},
		`
		SELECT $columns
		FROM auth_user
		LEFT JOIN handmade_asset AS auth_user_avatar ON auth_user_avatar.id = auth_user.avatar_asset_id
		WHERE
			auth_user.id = ANY($1)
		`,
		userIds,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch users for projects")
	}

	// Build the final result set with real user data
	res := make([]ProjectOwners, len(projectIds))
	for i, pid := range projectIds {
		res[i] = ProjectOwners{ProjectID: pid}
	}
	for _, iuserproject := range iuserprojects {
		userProject := iuserproject.(*userProject)

		// Get a pointer to the existing record in the result
		var projectOwners *ProjectOwners
		for i := range res {
			if res[i].ProjectID == userProject.ProjectID {
				projectOwners = &res[i]
			}
		}

		// Get the full user record we fetched
		var user *models.User
		for _, iuser := range iusers {
			u := iuser.(*userQuery).User
			if u.ID == userProject.UserID {
				user = &u
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
	perf := perf.ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch owners for project")
	defer perf.EndBlock()

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
		itag, err := db.QueryOne(ctx, tx, models.Tag{},
			`
			INSERT INTO tags (text) VALUES ($1)
			RETURNING $columns
			`,
			tagText,
		)
		if err != nil {
			return nil, oops.New(err, "failed to create new tag for project")
		}
		resultTag = itag.(*models.Tag)

		// Attach it to the project
		_, err = tx.Exec(ctx,
			`
			UPDATE handmade_project
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
		itag, err := db.QueryOne(ctx, tx, models.Tag{},
			`
			UPDATE tags
			SET text = $1
			WHERE id = (SELECT tag FROM handmade_project WHERE id = $2)
			RETURNING $columns
			`,
			tagText, projectID,
		)
		if err != nil {
			return nil, oops.New(err, "failed to update existing tag")
		}
		resultTag = itag.(*models.Tag)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to commit transaction")
	}

	return resultTag, nil
}

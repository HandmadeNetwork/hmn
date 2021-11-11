package website

import (
	"context"

	"git.handmade.network/hmn/hmn/src/hmnurl"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

type ProjectTypeQuery int

const (
	PersonalProjects ProjectTypeQuery = 1 << iota
	OfficialProjects
)

type ProjectsQuery struct {
	// Available on all project queries
	Lifecycles    []models.ProjectLifecycle // if empty, defaults to models.VisibleProjectLifecycles
	Types         ProjectTypeQuery          // bitfield
	IncludeHidden bool

	// Ignored when using FetchProject
	ProjectIDs []int    // if empty, all projects
	Slugs      []string // if empty, all projects

	// Ignored when using CountProjects
	Limit, Offset int // if empty, no pagination
}

type ProjectAndStuff struct {
	Project models.Project
	Owners  []*models.User
}

func FetchProjects(
	ctx context.Context,
	dbConn db.ConnOrTx,
	currentUser *models.User,
	q ProjectsQuery,
) ([]ProjectAndStuff, error) {
	perf := ExtractPerf(ctx)
	perf.StartBlock("SQL", "Fetch projects")
	defer perf.EndBlock()

	var currentUserID *int
	if currentUser != nil {
		currentUserID = &currentUser.ID
	}

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return nil, oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	// Fetch all valid projects (not yet subject to user permission checks)
	var qb db.QueryBuilder
	qb.Add(`
		SELECT $columns
		FROM
			handmade_project AS project
		WHERE
			TRUE
	`)
	if !q.IncludeHidden {
		qb.Add(`AND NOT hidden`)
	}
	if len(q.ProjectIDs) > 0 {
		qb.Add(`AND project.id = ANY ($?)`, q.ProjectIDs)
	}
	if len(q.Slugs) > 0 {
		qb.Add(`AND (project.slug != '' AND project.slug = ANY ($?))`, q.Slugs)
	}
	if len(q.Lifecycles) > 0 {
		qb.Add(`AND project.lifecycle = ANY($?)`, q.Lifecycles)
	} else {
		qb.Add(`AND project.lifecycle = ANY($?)`, models.VisibleProjectLifecycles)
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
	if q.Limit > 0 {
		qb.Add(`LIMIT $? OFFSET $?`, q.Limit, q.Offset)
	}
	itProjects, err := db.Query(ctx, dbConn, models.Project{}, qb.String(), qb.Args()...)
	if err != nil {
		return nil, oops.New(err, "failed to fetch projects")
	}
	iprojects := itProjects.ToSlice()

	// Fetch project owners to do permission checks
	projectIds := make([]int, len(iprojects))
	for i, iproject := range iprojects {
		projectIds[i] = iproject.(*models.Project).ID
	}
	projectOwners, err := FetchMultipleProjectsOwners(ctx, tx, projectIds)
	if err != nil {
		return nil, err
	}

	var res []ProjectAndStuff
	for i, iproject := range iprojects {
		project := iproject.(*models.Project)
		owners := projectOwners[i].Owners

		/*
			Per our spec, a user can see a project if:
			- The project is official
			- The project is personal and all of the project's owners are approved
			- The project is personal and the current user is a collaborator (regardless of user status)

			See https://www.notion.so/handmade-network/Technical-Plan-a11aaa9ea2f14d9a95f7d7780edd789c
		*/

		var projectVisible bool
		if project.Personal {
			allOwnersApproved := true
			for _, owner := range owners {
				if owner.Status != models.UserStatusApproved {
					allOwnersApproved = false
				}
				if currentUserID != nil && *currentUserID == owner.ID {
					projectVisible = true
				}
			}
			if allOwnersApproved {
				projectVisible = true
			}
		} else {
			projectVisible = true
		}

		if projectVisible {
			res = append(res, ProjectAndStuff{
				Project: *project,
				Owners:  owners,
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
	perf := ExtractPerf(ctx)
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

func CanEditProject(c *RequestContext, user *models.User, projectId int) (bool, error) {
	if user != nil {
		if user.IsStaff {
			return true, nil
		} else {
			owners, err := FetchProjectOwners(c.Context(), c.Conn, projectId)
			if err != nil {
				return false, err
			}
			for _, owner := range owners {
				if owner.ID == user.ID {
					return true, nil
				}
			}
		}
	}
	return false, nil
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
	perf := ExtractPerf(ctx)
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
	it, err := db.Query(ctx, tx, userProject{},
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
	iuserprojects := it.ToSlice()

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
	it, err = db.Query(ctx, tx, models.User{},
		`
		SELECT $columns
		FROM auth_user
		WHERE
			id = ANY($1)
		`,
		userIds,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch users for projects")
	}
	iusers := it.ToSlice()

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
			u := iuser.(*models.User)
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
	perf := ExtractPerf(ctx)
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

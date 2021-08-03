package website

import (
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

func CanEditProject(c *RequestContext, user *models.User, projectId int) (bool, error) {
	if user != nil {
		if user.IsStaff {
			return true, nil
		} else {
			owners, err := FetchProjectOwners(c, projectId)
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

func FetchProjectOwners(c *RequestContext, projectId int) ([]*models.User, error) {
	var result []*models.User
	c.Perf.StartBlock("SQL", "Fetching project owners")
	type ownerQuery struct {
		Owner models.User `db:"auth_user"`
	}
	ownerQueryResult, err := db.Query(c.Context(), c.Conn, ownerQuery{},
		`
		SELECT $columns
		FROM
			auth_user
			INNER JOIN handmade_user_projects AS uproj ON uproj.user_id = auth_user.id
		WHERE
			uproj.project_id = $1
		`,
		projectId,
	)
	c.Perf.EndBlock()
	if err != nil {
		return result, oops.New(err, "failed to fetch owners for project")
	}
	for _, ownerRow := range ownerQueryResult.ToSlice() {
		result = append(result, &ownerRow.(*ownerQuery).Owner)
	}
	return result, nil
}

package models

type Follow struct {
	UserID             int  `db:"user_id"`
	FollowingUserID    *int `db:"following_user_id"`
	FollowingProjectID *int `db:"following_project_id"`
}

package models

import "time"

type JamProject struct {
	ProjectID     int    `db:"project_id"`
	JamSlug       string `db:"jam_slug"`
	Participating bool   `db:"participating"`
	JamName       string
	JamStartTime  time.Time
}

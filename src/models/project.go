package models

import (
	"reflect"
	"time"
)

const (
	HMNProjectID   = 1
	HMNProjectSlug = "hmn"
)

var ProjectType = reflect.TypeOf(Project{})

type ProjectLifecycle int

const (
	ProjectLifecycleUnapproved = iota
	ProjectLifecycleApprovalRequired
	ProjectLifecycleActive
	ProjectLifecycleHiatus
	ProjectLifecycleDead
	ProjectLifecycleLTSRequired
	ProjectLifecycleLTS
)

type Project struct {
	ID int `db:"id"`

	ForumID *int `db:"forum_id"`

	Slug        string `db:"slug"`
	Name        string `db:"name"`
	Blurb       string `db:"blurb"`
	Description string `db:"description"`

	Lifecycle ProjectLifecycle `db:"lifecycle"` // TODO(asaf): Ensure we only fetch projects in the correct lifecycle phase everywhere.

	Color1 string `db:"color_1"`
	Color2 string `db:"color_2"`

	AllLastUpdated time.Time `db:"all_last_updated"`
}

func (p *Project) IsHMN() bool {
	return p.ID == HMNProjectID
}

func (p *Project) Subdomain() string {
	if p.IsHMN() {
		return ""
	}

	return p.Slug
}

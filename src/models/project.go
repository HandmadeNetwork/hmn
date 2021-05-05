package models

import (
	"reflect"
	"time"
)

const HMNProjectID = 1

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

	Slug        *string `db:"slug"` // TODO: Migrate these to NOT NULL
	Name        *string `db:"name"`
	Blurb       *string `db:"blurb"`
	Description *string `db:"description"`

	Lifecycle ProjectLifecycle `db:"lifecycle"`

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

	return *p.Slug
}

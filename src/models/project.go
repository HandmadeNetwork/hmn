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

// NOTE(asaf): Just checking the lifecycle is not sufficient. Visible projects also must have flags = 0.
var VisibleProjectLifecycles = []ProjectLifecycle{
	ProjectLifecycleActive,
	ProjectLifecycleHiatus,
	ProjectLifecycleLTSRequired, // NOTE(asaf): LTS means complete
	ProjectLifecycleLTS,
}

const RecentProjectUpdateTimespanSec = 60 * 60 * 24 * 28 // NOTE(asaf): Four weeks

type Project struct {
	ID int `db:"id"`

	ForumID *int `db:"forum_id"`

	Slug              string `db:"slug"`
	Name              string `db:"name"`
	Blurb             string `db:"blurb"`
	Description       string `db:"description"`
	ParsedDescription string `db:"descparsed"`

	Lifecycle ProjectLifecycle `db:"lifecycle"` // TODO(asaf): Ensure we only fetch projects in the correct lifecycle phase everywhere.

	Color1 string `db:"color_1"`
	Color2 string `db:"color_2"`

	LogoLight string `db:"logolight"`
	LogoDark  string `db:"logodark"`

	Flags          int       `db:"flags"` // NOTE(asaf): Flags is currently only used to mark a project as hidden. Flags == 1 means hidden. Flags == 0 means visible.
	Featured       bool      `db:"featured"`
	DateApproved   time.Time `db:"date_approved"`
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

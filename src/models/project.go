package models

import (
	"reflect"
	"time"
)

const HMNProjectID = 1

var ProjectType = reflect.TypeOf(Project{})

type Project struct {
	ID int `db:"id"`

	Slug        *string `db:"slug"` // TODO: Migrate these to NOT NULL
	Name        *string `db:"name"`
	Blurb       *string `db:"blurb"`
	Description *string `db:"description"`

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

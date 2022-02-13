package models

import (
	"reflect"
	"regexp"
	"strings"
	"time"
)

const (
	HMNProjectID   = 1
	HMNProjectSlug = "hmn"
)

var ProjectType = reflect.TypeOf(Project{})

type ProjectLifecycle int

const (
	ProjectLifecycleUnapproved ProjectLifecycle = iota
	ProjectLifecycleApprovalRequired
	ProjectLifecycleActive
	ProjectLifecycleHiatus
	ProjectLifecycleDead
	ProjectLifecycleLTSRequired
	ProjectLifecycleLTS
)

var AllProjectLifecycles = []ProjectLifecycle{
	ProjectLifecycleUnapproved,
	ProjectLifecycleApprovalRequired,
	ProjectLifecycleActive,
	ProjectLifecycleHiatus,
	ProjectLifecycleDead,
	ProjectLifecycleLTSRequired,
	ProjectLifecycleLTS,
}

// NOTE(asaf): Just checking the lifecycle is not sufficient. Visible projects also must not be hidden.
var VisibleProjectLifecycles = []ProjectLifecycle{
	ProjectLifecycleActive,
	ProjectLifecycleHiatus,
	ProjectLifecycleLTSRequired, // NOTE(asaf): LTS means complete
	ProjectLifecycleLTS,
}

func (lc ProjectLifecycle) In(lcs []ProjectLifecycle) bool {
	for _, v := range lcs {
		if lc == v {
			return true
		}
	}
	return false
}

const RecentProjectUpdateTimespanSec = 60 * 60 * 24 * 28 // NOTE(asaf): Four weeks

type Project struct {
	ID int `db:"id"`

	ForumID *int `db:"forum_id"`

	Slug              string `db:"slug"`
	Name              string `db:"name"`
	TagID             *int   `db:"tag"`
	Blurb             string `db:"blurb"`
	Description       string `db:"description"`
	ParsedDescription string `db:"descparsed"`

	Lifecycle ProjectLifecycle `db:"lifecycle"` // TODO(asaf): Ensure we only fetch projects in the correct lifecycle phase everywhere.

	Color1 string `db:"color_1"`
	Color2 string `db:"color_2"`

	Personal              bool      `db:"personal"`
	Hidden                bool      `db:"hidden"`
	Featured              bool      `db:"featured"`
	DateApproved          time.Time `db:"date_approved"`
	AllLastUpdated        time.Time `db:"all_last_updated"`
	ForumLastUpdated      time.Time `db:"forum_last_updated"`
	BlogLastUpdated       time.Time `db:"blog_last_updated"`
	AnnotationLastUpdated time.Time `db:"annotation_last_updated"`

	ForumEnabled   bool `db:"forum_enabled"`
	BlogEnabled    bool `db:"blog_enabled"`
	LibraryEnabled bool `db:"library_enabled"` // TODO: Delete this field from the db
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

// Checks whether the project has forums enabled. This should restrict the creation of new forum
// content, but it should NOT prevent the viewing of existing forum content. (Projects may at one
// point have forums enabled, write some stuff, and then later disable forums, and we want that
// content to stay accessible.) Hiding the navigation is ok.
func (p *Project) HasForums() bool {
	return !p.Personal && p.ForumEnabled
}

// Same as HasForums, but for blogs.
func (p *Project) HasBlog() bool {
	return !p.Personal && p.BlogEnabled
}

var slugUnsafeChars = regexp.MustCompile(`[^a-zA-Z0-9-]`)
var slugHyphenRun = regexp.MustCompile(`-+`)

// Generates a URL-safe version of a personal project's name.
func GeneratePersonalProjectSlug(name string) string {
	slug := name
	slug = slugUnsafeChars.ReplaceAllLiteralString(slug, "-")
	slug = slugHyphenRun.ReplaceAllLiteralString(slug, "-")
	slug = strings.Trim(slug, "-")
	slug = strings.ToLower(slug)

	return slug
}

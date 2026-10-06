package models

type BetaWeekProjectSubmission struct {
	ID        int    `db:"id"`
	EventSlug string `db:"event_slug"`

	ProjectID           int    `db:"project_id"`
	ProjectStatus       string `db:"project_status"`
	ProjectPlatforms    string `db:"project_platforms"`
	ProjectInstructions string `db:"project_instructions"`
	AuthorAvailability  string `db:"author_availability"`
	AuthorGoals         string `db:"author_goals"`
}

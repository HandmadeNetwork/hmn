package models

import (
	"time"

	"github.com/google/uuid"
)

type Podcast struct {
	ID        int `db:"id"`
	ImageID   int `db:"image_id"`
	ProjectID int `db:"project_id"`

	Title       string `db:"title"`
	Description string `db:"description"`
	Language    string `db:"language"`
}

type PodcastEpisode struct {
	GUID      uuid.UUID `db:"guid"`
	PodcastID int       `db:"podcast_id"`

	Title           string    `db:"title"`
	Description     string    `db:"description"`
	DescriptionHtml string    `db:"description_rendered"`
	AudioFile       string    `db:"audio_filename"`
	PublicationDate time.Time `db:"pub_date"`
	Duration        int       `db:"duration"` // NOTE(asaf): In seconds
	EpisodeNumber   int       `db:"episode_number"`
	SeasonNumber    *int      `db:"season_number"` // TODO(asaf): Do we need this??
}

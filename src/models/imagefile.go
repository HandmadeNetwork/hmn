package models

type ImageFile struct {
	ID        int    `db:"id"`
	File      string `db:"file"` // relative to public/media
	Size      int    `db:"size"`
	Sha1Sum   string `db:"sha1sum"`
	Protected bool   `db:"protected"`
	Height    int    `db:"height"`
	Width     int    `db:"width"`
}

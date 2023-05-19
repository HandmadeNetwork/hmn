package models

import (
	"github.com/google/uuid"
)

type Asset struct {
	ID         uuid.UUID `db:"id"`
	UploaderID *int      `db:"uploader_id"`

	S3Key          string `db:"s3_key"`
	ThumbnailS3Key string `db:"thumbnail_s3_key"`
	Filename       string `db:"filename"`
	Size           int    `db:"size"`
	MimeType       string `db:"mime_type"`
	Sha1Sum        string `db:"sha1sum"`
	Width          int    `db:"width"`
	Height         int    `db:"height"`
}

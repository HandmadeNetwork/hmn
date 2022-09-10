package models

import (
	"time"
)

type EduArticle struct {
	ID int `db:"id"`

	Title       string `db:"title"`
	Slug        string `db:"slug"`
	Description string `db:"description"`
	Published   bool   `db:"published"` // Unpublished articles are visible to authors and beta testers.

	Type EduArticleType `db:"type"`

	CurrentVersionID int                `db:"current_version"`
	CurrentVersion   *EduArticleVersion // not in DB, set by helpers
}

type EduArticleType int

const (
	EduArticleTypeArticle EduArticleType = iota + 1
	EduArticleTypeGlossary
)

type EduArticleVersion struct {
	ID        int       `db:"id"`
	ArticleID int       `db:"article_id"`
	Date      time.Time `db:"date"`
	EditorID  *int      `db:"editor_id"`

	ContentRaw  string `db:"content_raw"`
	ContentHTML string `db:"content_html"`
}

type EduRole int

const (
	EduRoleNone EduRole = iota
	EduRoleBeta
	EduRoleAuthor
)

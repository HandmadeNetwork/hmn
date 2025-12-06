package models

import "regexp"

type Tag struct {
	ID   int    `db:"id"`
	Text string `db:"text"`
}

var REValidTag = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

func ValidateTagText(text string) bool {
	if text == "" {
		return true
	}

	if len(text) > 20 {
		return false
	}
	if !REValidTag.MatchString(text) {
		return false
	}

	return true
}

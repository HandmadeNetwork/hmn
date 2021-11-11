package models

type Tag struct {
	ID   int    `db:"id"`
	Text string `db:"text"`
}

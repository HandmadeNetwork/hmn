package models

import "time"

type Session struct {
	ID        string    `db:"id"`
	Username  string    `db:"username"`
	ExpiresAt time.Time `db:"expires_at"`
}

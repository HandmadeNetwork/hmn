package models

import "time"

type EmailBlacklist struct {
	Email         string    `db:"email"`
	BlacklistedAt time.Time `db:"blacklisted_at"`
	BouncedAt     time.Time `db:"bounced_at"`
	Reason        string    `db:"reason"`
	Details       string    `db:"details"`
}

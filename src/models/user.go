package models

import "time"

type User struct {
	ID int `db:"id"`

	Username string `db:"username"`
	Password string `db:"password"`
	Email    string `db:"email"`

	DateJoined time.Time  `db:"date_joined"`
	LastLogin  *time.Time `db:"last_login"`

	IsSuperuser bool `db:"is_superuser"`
	IsStaff     bool `db:"is_staff"`
	IsActive    bool `db:"is_active"`
}

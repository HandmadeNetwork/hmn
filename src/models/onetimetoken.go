package models

import (
	"time"

	"github.com/google/uuid"
)

type OneTimeTokenType int

const (
	TokenTypeRegistration OneTimeTokenType = iota + 1
	TokenTypePasswordReset
)

type OneTimeToken struct {
	ID      int              `db:"id"`
	OwnerID int              `db:"owner_id"`
	Type    OneTimeTokenType `db:"token_type`
	Created time.Time        `db:"created"`
	Expires time.Time        `db:"expires"`
	Content string           `db:"token_content"`
}

func GenerateToken() string {
	return uuid.New().String()
}

package models

import (
	"reflect"
	"time"

	"github.com/google/uuid"
)

var UserType = reflect.TypeOf(User{})

type UserStatus int

const (
	UserStatusInactive  UserStatus = 1 // Default for new users
	UserStatusConfirmed            = 2 // Confirmed email address
	UserStatusApproved             = 3 // Approved by an admin and allowed to publicly post
	UserStatusBanned               = 4 // BALEETED
)

type User struct {
	ID int `db:"id"`

	Username string `db:"username"`
	Password string `db:"password"`
	Email    string `db:"email"`

	DateJoined time.Time  `db:"date_joined"`
	LastLogin  *time.Time `db:"last_login"`

	IsStaff bool       `db:"is_staff"`
	Status  UserStatus `db:"status"`

	Name          string     `db:"name"`
	Bio           string     `db:"bio"`
	Blurb         string     `db:"blurb"`
	Signature     string     `db:"signature"`
	AvatarAssetID *uuid.UUID `db:"avatar_asset_id"`

	DarkTheme bool   `db:"darktheme"`
	Timezone  string `db:"timezone"`

	ShowEmail      bool `db:"showemail"`
	CanEditLibrary bool `db:"edit_library"`

	DiscordSaveShowcase                 bool `db:"discord_save_showcase"`
	DiscordDeleteSnippetOnMessageDelete bool `db:"discord_delete_snippet_on_message_delete"`

	MarkedAllReadAt time.Time `db:"marked_all_read_at"`

	// Non-db fields, to be filled in by fetch helpers
	AvatarAsset *Asset
}

func (u *User) BestName() string {
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

func (u *User) IsActive() bool {
	return u.Status == UserStatusConfirmed
}

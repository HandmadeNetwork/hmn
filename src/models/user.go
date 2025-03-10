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
	UserStatusConfirmed UserStatus = 2 // Confirmed email address
	UserStatusApproved  UserStatus = 3 // Approved by an admin and allowed to publicly post
	UserStatusBanned    UserStatus = 4 // BALEETED
)

type User struct {
	ID int `db:"id"`

	Username string `db:"username"`
	Password string `db:"password"`
	Email    string `db:"email"`

	DateJoined time.Time  `db:"date_joined"`
	LastLogin  *time.Time `db:"last_login"`

	IsStaff       bool       `db:"is_staff"`
	Status        UserStatus `db:"status"`
	EducationRole EduRole    `db:"education_role"`
	Featured      bool       `db:"featured"`

	Name          string     `db:"name"`
	Bio           string     `db:"bio"`
	Blurb         string     `db:"blurb"`
	Signature     string     `db:"signature"`
	AvatarAssetID *uuid.UUID `db:"avatar_asset_id"`

	Timezone string `db:"timezone"`

	ShowEmail bool `db:"showemail"`

	DiscordSaveShowcase                 bool `db:"discord_save_showcase"`
	DiscordDeleteSnippetOnMessageDelete bool `db:"discord_delete_snippet_on_message_delete"`

	MarkedAllReadAt time.Time `db:"marked_all_read_at"`

	// Non-db fields, to be filled in by fetch helpers
	AvatarAsset *Asset
	DiscordUser *DiscordUser

	BanReason string `db:"ban_reason"`
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

func (u *User) CanSeeUnpublishedEducationContent() bool {
	return u.IsStaff || u.EducationRole == EduRoleBeta || u.EducationRole == EduRoleAuthor
}

func (u *User) CanAuthorEducation() bool {
	return u.IsStaff || u.EducationRole == EduRoleAuthor
}

type PendingLogin struct {
	ID             string    `db:"id"`
	ExpiresAt      time.Time `db:"expires_at"`
	DestinationUrl string    `db:"destination_url"`
}

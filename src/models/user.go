package models

import (
	"reflect"
	"time"
)

var UserType = reflect.TypeOf(User{})

type User struct {
	ID int `db:"id"`

	Username string `db:"username"`
	Password string `db:"password"`
	Email    string `db:"email"`

	DateJoined time.Time  `db:"date_joined"`
	LastLogin  *time.Time `db:"last_login"`

	IsStaff  bool `db:"is_staff"`
	IsActive bool `db:"is_active"`

	Name      string  `db:"name"`
	Bio       string  `db:"bio"`
	Blurb     string  `db:"blurb"`
	Signature string  `db:"signature"`
	Avatar    *string `db:"avatar"` // TODO: Image field stuff?

	DarkTheme     bool   `db:"darktheme"`
	Timezone      string `db:"timezone"`
	ProfileColor1 string `db:"color_1"`
	ProfileColor2 string `db:"color_2"`

	ShowEmail      bool `db:"showemail"`
	CanEditLibrary bool `db:"edit_library"`

	DiscordSaveShowcase                 bool `db:"discord_save_showcase"`
	DiscordDeleteSnippetOnMessageDelete bool `db:"discord_delete_snippet_on_message_delete"`
}

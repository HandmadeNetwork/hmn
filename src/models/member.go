package models

type Member struct {
	UserID int

	Name      *string `db:"name"` // TODO: Migrate to not null
	Bio       *string `db:"bio"`
	Blurb     *string `db:"blurb"`
	Signature *string `db:"signature"`
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

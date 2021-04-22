package templates

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
)

func PostToTemplate(p *models.Post) Post {
	return Post{
		Preview:  p.Preview,
		ReadOnly: p.ReadOnly,
	}
}

func ProjectToTemplate(p *models.Project) Project {
	return Project{
		Name:      maybeString(p.Name),
		Subdomain: p.Subdomain(),
		Color1:    p.Color1,
		Color2:    p.Color2,

		IsHMN: p.IsHMN(),

		HasBlog:    true, // TODO: Check flag sets or whatever
		HasForum:   true,
		HasWiki:    true,
		HasLibrary: true,
	}
}

func UserToTemplate(u *models.User) User {
	avatar := ""
	if u.Avatar != nil {
		avatar = hmnurl.StaticUrl(*u.Avatar, nil)
	}

	name := u.Name
	if u.Name == "" {
		name = u.Username
	}

	return User{
		Username:    u.Username,
		Email:       u.Email,
		IsSuperuser: u.IsSuperuser,
		IsStaff:     u.IsStaff,

		Name:       name,
		Blurb:      u.Blurb,
		Signature:  u.Signature,
		AvatarUrl:  avatar, // TODO
		ProfileUrl: hmnurl.Url("m/"+u.Username, nil),

		DarkTheme:     u.DarkTheme,
		Timezone:      u.Timezone,
		ProfileColor1: u.ProfileColor1,
		ProfileColor2: u.ProfileColor2,

		CanEditLibrary:                      u.CanEditLibrary,
		DiscordSaveShowcase:                 u.DiscordSaveShowcase,
		DiscordDeleteSnippetOnMessageDelete: u.DiscordDeleteSnippetOnMessageDelete,
	}
}

func maybeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

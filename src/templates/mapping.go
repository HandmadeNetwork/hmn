package templates

import "git.handmade.network/hmn/hmn/src/models"

func PostToTemplate(p *models.Post) Post {
	return Post{
		Preview:  p.Preview,
		ReadOnly: p.ReadOnly,
	}
}

func ProjectToTemplate(p *models.Project) Project {
	return Project{
		Name:      maybeString(p.Name),
		Subdomain: maybeString(p.Slug),
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
	return User{
		Username:    u.Username,
		Email:       u.Email,
		IsSuperuser: u.IsSuperuser,
		IsStaff:     u.IsStaff,

		Name:      u.Name,
		Blurb:     u.Blurb,
		Signature: u.Signature,

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

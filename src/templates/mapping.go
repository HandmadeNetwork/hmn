package templates

import "git.handmade.network/hmn/hmn/src/models"

func MemberToTemplate(m *models.Member) Member {
	return Member{
		Name:      maybeString(m.Name),
		Blurb:     maybeString(m.Blurb),
		Signature: maybeString(m.Signature),

		DarkTheme:     m.DarkTheme,
		Timezone:      m.Timezone,
		ProfileColor1: m.ProfileColor1,
		ProfileColor2: m.ProfileColor2,

		CanEditLibrary:                      m.CanEditLibrary,
		DiscordSaveShowcase:                 m.DiscordSaveShowcase,
		DiscordDeleteSnippetOnMessageDelete: m.DiscordDeleteSnippetOnMessageDelete,
	}
}

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
	}
}

func maybeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

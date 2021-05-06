package templates

import (
	"html/template"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
)

func PostToTemplate(p *models.Post, author *models.User) Post {
	var authorUser *User
	if author != nil {
		authorTmpl := UserToTemplate(author)
		authorUser = &authorTmpl
	}

	return Post{
		ID:  p.ID,
		Url: "nope", // TODO

		Preview:  p.Preview,
		ReadOnly: p.ReadOnly,

		Author: authorUser,
		// No content. Do it yourself if you care.
		PostDate: p.PostDate,

		IP: p.IP.String(),
	}
}

func PostToTemplateWithContent(p *models.Post, author *models.User, content string) Post {
	post := PostToTemplate(p, author)
	post.Content = template.HTML(content)

	return post
}

func ProjectToTemplate(p *models.Project) Project {
	return Project{
		Name:      p.Name,
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

func ThreadToTemplate(t *models.Thread) Thread {
	return Thread{
		Title:  t.Title,
		Locked: t.Locked,
		Sticky: t.Sticky,
	}
}

func UserToTemplate(u *models.User) User {
	// TODO: Handle deleted users. Maybe not here, but if not, at call sites of this function.

	avatar := ""
	if u.Avatar != nil {
		avatar = hmnurl.StaticUrl(*u.Avatar, nil)
	}

	name := u.Name
	if u.Name == "" {
		name = u.Username
	}

	return User{
		ID:          u.ID,
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

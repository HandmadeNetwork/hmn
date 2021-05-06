package templates

import (
	"html/template"
	"net"

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
		ID: p.ID,

		// Urls not set here. See AddUrls.

		Preview:  p.Preview,
		ReadOnly: p.ReadOnly,

		Author: authorUser,
		// No content. A lot of the time we don't have this handy and don't need it. See AddContentVersion.
		PostDate: p.PostDate,

		IP: p.IP.String(),
	}
}

func (p *Post) AddContentVersion(ver models.PostVersion, editor *models.User) {
	p.Content = template.HTML(ver.TextParsed)

	if editor != nil {
		editorTmpl := UserToTemplate(editor)
		p.Editor = &editorTmpl
		p.EditDate = ver.EditDate
		p.EditIP = maybeIp(ver.EditIP)
		p.EditReason = ver.EditReason
	}
}

func (p *Post) AddUrls(projectSlug string, subforums []string, threadId int, postId int) {
	p.Url = hmnurl.BuildForumPost(projectSlug, subforums, threadId, postId)
	p.DeleteUrl = hmnurl.BuildForumPostDelete(projectSlug, subforums, threadId, postId)
	p.EditUrl = hmnurl.BuildForumPostEdit(projectSlug, subforums, threadId, postId)
	p.ReplyUrl = hmnurl.BuildForumPostReply(projectSlug, subforums, threadId, postId)
	p.QuoteUrl = hmnurl.BuildForumPostQuote(projectSlug, subforums, threadId, postId)
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

func maybeIp(ip *net.IPNet) string {
	if ip == nil {
		return ""
	}

	return ip.String()
}

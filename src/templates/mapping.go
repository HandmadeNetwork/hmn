package templates

import (
	"html/template"
	"net"
	"regexp"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
)

func PostToTemplate(p *models.Post, author *models.User, currentTheme string) Post {
	var authorUser *User
	if author != nil {
		authorTmpl := UserToTemplate(author, currentTheme)
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

func (p *Post) AddContentVersion(ver models.PostVersion, editor *models.User, currentTheme string) {
	p.Content = template.HTML(ver.TextParsed)

	if editor != nil {
		editorTmpl := UserToTemplate(editor, currentTheme)
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

var LifecycleBadgeClasses = map[models.ProjectLifecycle]string{
	models.ProjectLifecycleUnapproved:       "",
	models.ProjectLifecycleApprovalRequired: "",
	models.ProjectLifecycleActive:           "",
	models.ProjectLifecycleHiatus:           "notice-hiatus",
	models.ProjectLifecycleDead:             "notice-dead",
	models.ProjectLifecycleLTSRequired:      "",
	models.ProjectLifecycleLTS:              "notice-lts",
}

var LifecycleBadgeStrings = map[models.ProjectLifecycle]string{
	models.ProjectLifecycleUnapproved:       "",
	models.ProjectLifecycleApprovalRequired: "",
	models.ProjectLifecycleActive:           "",
	models.ProjectLifecycleHiatus:           "On Hiatus",
	models.ProjectLifecycleDead:             "Dead",
	models.ProjectLifecycleLTSRequired:      "",
	models.ProjectLifecycleLTS:              "Complete",
}

func ProjectUrl(p *models.Project) string {
	var url string
	if p.Lifecycle == models.ProjectLifecycleUnapproved || p.Lifecycle == models.ProjectLifecycleApprovalRequired {
		url = hmnurl.BuildProjectNotApproved(p.Slug)
	} else {
		url = hmnurl.BuildProjectHomepage(p.Slug)
	}
	return url
}

func ProjectToTemplate(p *models.Project, theme string) Project {
	logo := p.LogoLight
	if theme == "dark" {
		logo = p.LogoDark
	}
	url := ProjectUrl(p)
	return Project{
		Name:              p.Name,
		Subdomain:         p.Subdomain(),
		Color1:            p.Color1,
		Color2:            p.Color2,
		Url:               url,
		Blurb:             p.Blurb,
		ParsedDescription: template.HTML(p.ParsedDescription),

		Logo: hmnurl.BuildUserFile(logo),

		LifecycleBadgeClass: LifecycleBadgeClasses[p.Lifecycle],
		LifecycleString:     LifecycleBadgeStrings[p.Lifecycle],

		IsHMN: p.IsHMN(),

		HasBlog:    true, // TODO: Check flag sets or whatever
		HasForum:   true,
		HasWiki:    true,
		HasLibrary: true,

		DateApproved: p.DateApproved,
	}
}

func SessionToTemplate(s *models.Session) Session {
	return Session{
		CSRFToken: s.CSRFToken,
	}
}

func ThreadToTemplate(t *models.Thread) Thread {
	return Thread{
		Title:  t.Title,
		Locked: t.Locked,
		Sticky: t.Sticky,
	}
}

func UserAvatarUrl(u *models.User, currentTheme string) string {
	if currentTheme == "" {
		currentTheme = "light"
	}
	avatar := ""
	if u.Avatar != nil && len(*u.Avatar) > 0 {
		avatar = hmnurl.BuildUserFile(*u.Avatar)
	} else {
		avatar = hmnurl.BuildTheme("empty-avatar.svg", currentTheme, true)
	}
	return avatar
}

func UserDisplayName(u *models.User) string {
	name := u.Name
	if u.Name == "" {
		name = u.Username
	}
	return name
}

func UserToTemplate(u *models.User, currentTheme string) User {
	// TODO: Handle deleted users. Maybe not here, but if not, at call sites of this function.

	email := ""
	if u.ShowEmail {
		// TODO(asaf): Always show email to admins
		email = u.Email
	}

	return User{
		ID:          u.ID,
		Username:    u.Username,
		Email:       email,
		IsSuperuser: u.IsSuperuser,
		IsStaff:     u.IsStaff,

		Name:       UserDisplayName(u),
		Blurb:      u.Blurb,
		Signature:  u.Signature,
		DateJoined: u.DateJoined,
		AvatarUrl:  UserAvatarUrl(u, currentTheme),
		ProfileUrl: hmnurl.BuildUserProfile(u.Username),

		DarkTheme:     u.DarkTheme,
		Timezone:      u.Timezone,
		ProfileColor1: u.ProfileColor1,
		ProfileColor2: u.ProfileColor2,

		CanEditLibrary:                      u.CanEditLibrary,
		DiscordSaveShowcase:                 u.DiscordSaveShowcase,
		DiscordDeleteSnippetOnMessageDelete: u.DiscordDeleteSnippetOnMessageDelete,
	}
}

var RegexServiceYoutube = regexp.MustCompile(`youtube\.com/(c/)?(?P<userdata>[\w/-]+)$`)
var RegexServiceTwitter = regexp.MustCompile(`twitter\.com/(?P<userdata>\w+)$`)
var RegexServiceGithub = regexp.MustCompile(`github\.com/(?P<userdata>[\w/-]+)$`)
var RegexServiceTwitch = regexp.MustCompile(`twitch\.tv/(?P<userdata>[\w/-]+)$`)
var RegexServiceHitbox = regexp.MustCompile(`hitbox\.tv/(?P<userdata>[\w/-]+)$`)
var RegexServicePatreon = regexp.MustCompile(`patreon\.com/(?P<userdata>[\w/-]+)$`)
var RegexServiceSoundcloud = regexp.MustCompile(`soundcloud\.com/(?P<userdata>[\w/-]+)$`)
var RegexServiceItch = regexp.MustCompile(`(?P<userdata>[\w/-]+)\.itch\.io/?$`)

var LinkServiceMap = map[string]*regexp.Regexp{
	"youtube":    RegexServiceYoutube,
	"twitter":    RegexServiceTwitter,
	"github":     RegexServiceGithub,
	"twitch":     RegexServiceTwitch,
	"hitbox":     RegexServiceHitbox,
	"patreon":    RegexServicePatreon,
	"soundcloud": RegexServiceSoundcloud,
	"itch":       RegexServiceItch,
}

func ParseKnownServicesForLink(link *models.Link) (serviceName string, userData string) {
	for name, re := range LinkServiceMap {
		match := re.FindStringSubmatch(link.Value)
		if match != nil {
			serviceName = name
			userData = match[re.SubexpIndex("userdata")]
			return
		}
	}
	return "", ""
}

func LinkToTemplate(link *models.Link) Link {
	name := ""
	if link.Name != nil {
		name = *link.Name
	}
	serviceName, serviceUserData := ParseKnownServicesForLink(link)
	return Link{
		Key:             link.Key,
		ServiceName:     serviceName,
		ServiceUserData: serviceUserData,
		Name:            name,
		Value:           link.Value,
	}
}

func TimelineItemsToJSON(items []TimelineItem) string {
	// NOTE(asaf): As of 2021-06-22: This only serializes the data necessary for snippet showcase.
	builder := strings.Builder{}
	builder.WriteRune('[')
	for i, item := range items {
		if i > 0 {
			builder.WriteRune(',')
		}
		builder.WriteRune('{')

		builder.WriteString(`"type":`)
		builder.WriteString(strconv.Itoa(int(item.Type)))
		builder.WriteRune(',')

		builder.WriteString(`"date":`)
		builder.WriteString(strconv.FormatInt(item.Date.UTC().Unix(), 10))
		builder.WriteRune(',')

		builder.WriteString(`"description":"`)
		jsonString := string(item.Description)
		jsonString = strings.ReplaceAll(jsonString, `\`, `\\`)
		jsonString = strings.ReplaceAll(jsonString, `"`, `\"`)
		jsonString = strings.ReplaceAll(jsonString, "\n", "\\n")
		jsonString = strings.ReplaceAll(jsonString, "\r", "\\r")
		jsonString = strings.ReplaceAll(jsonString, "\t", "\\t")
		builder.WriteString(jsonString)
		builder.WriteString(`",`)

		builder.WriteString(`"owner_name":"`)
		builder.WriteString(item.OwnerName)
		builder.WriteString(`",`)

		builder.WriteString(`"owner_avatar":"`)
		builder.WriteString(item.OwnerAvatarUrl)
		builder.WriteString(`",`)

		builder.WriteString(`"owner_url":"`)
		builder.WriteString(item.OwnerUrl)
		builder.WriteString(`",`)

		builder.WriteString(`"snippet_url":"`)
		builder.WriteString(item.Url)
		builder.WriteString(`",`)

		builder.WriteString(`"width":`)
		builder.WriteString(strconv.Itoa(item.Width))
		builder.WriteRune(',')

		builder.WriteString(`"height":`)
		builder.WriteString(strconv.Itoa(item.Height))
		builder.WriteRune(',')

		builder.WriteString(`"asset_url":"`)
		builder.WriteString(item.AssetUrl)
		builder.WriteString(`",`)

		builder.WriteString(`"discord_message_url":"`)
		builder.WriteString(item.DiscordMessageUrl)
		builder.WriteString(`"`)

		builder.WriteRune('}')
	}
	builder.WriteRune(']')
	return builder.String()
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

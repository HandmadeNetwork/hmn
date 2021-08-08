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
	return Post{
		ID: p.ID,

		// Urls not set here. They vary per thread type. Set 'em yourself!

		Preview:  p.Preview,
		ReadOnly: p.ReadOnly,

		Author: UserToTemplate(author, currentTheme),
		// No content. A lot of the time we don't have this handy and don't need it. See AddContentVersion.
		PostDate: p.PostDate,
	}
}

func (p *Post) AddContentVersion(ver models.PostVersion, editor *models.User) {
	p.Content = template.HTML(ver.TextParsed)
	p.IP = maybeIp(ver.IP)

	if editor != nil {
		editorTmpl := UserToTemplate(editor, "theme not required here")
		p.Editor = &editorTmpl
		p.EditDate = ver.Date
		p.EditReason = ver.EditReason
	}
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
	if u != nil && u.Avatar != nil && len(*u.Avatar) > 0 {
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
	if u == nil {
		return User{
			Name:      "Deleted user",
			AvatarUrl: UserAvatarUrl(u, currentTheme),
		}
	}

	email := ""
	if u.ShowEmail {
		// TODO(asaf): Always show email to admins
		email = u.Email
	}

	return User{
		ID:       u.ID,
		Username: u.Username,
		Email:    email,
		IsStaff:  u.IsStaff,

		Name:       UserDisplayName(u),
		Blurb:      u.Blurb,
		Signature:  u.Signature,
		DateJoined: u.DateJoined,
		AvatarUrl:  UserAvatarUrl(u, currentTheme),
		ProfileUrl: hmnurl.BuildUserProfile(u.Username),

		DarkTheme: u.DarkTheme,
		Timezone:  u.Timezone,

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
	/*
		// NOTE(asaf): While Name and Key are separate things, Name is almost always the same as Key in the db, which looks weird.
		//             So we're just going to ignore Name until we decide it's worth reusing.
		if link.Name != nil {
			name = *link.Name
		}
	*/
	serviceName, serviceUserData := ParseKnownServicesForLink(link)
	if serviceUserData != "" {
		name = serviceUserData
	}
	if name == "" {
		name = link.Value
	}
	return Link{
		Key:  link.Key,
		Name: name,
		Icon: serviceName,
		Url:  link.Value,
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
		builder.WriteString(item.OwnerName) // TODO: Do we need to do escaping on these other string fields too? Feels like someone could use this for XSS.
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

func PodcastToTemplate(projectSlug string, podcast *models.Podcast, imageFilename string) Podcast {
	imageUrl := ""
	if imageFilename != "" {
		imageUrl = hmnurl.BuildUserFile(imageFilename)
	}
	return Podcast{
		Title:       podcast.Title,
		Description: podcast.Description,
		Language:    podcast.Language,
		ImageUrl:    imageUrl,
		Url:         hmnurl.BuildPodcast(projectSlug),

		RSSUrl: hmnurl.BuildPodcastRSS(projectSlug),
		// TODO(asaf): Move this to the db if we want to support user podcasts
		AppleUrl:   "https://podcasts.apple.com/us/podcast/the-handmade-network-podcast/id1507790631",
		GoogleUrl:  "https://www.google.com/podcasts?feed=aHR0cHM6Ly9oYW5kbWFkZS5uZXR3b3JrL3BvZGNhc3QvcG9kY2FzdC54bWw%3D",
		SpotifyUrl: "https://open.spotify.com/show/2Nd9NjXscrBbQwYULiYKiU",
	}
}

func PodcastEpisodeToTemplate(projectSlug string, episode *models.PodcastEpisode, audioFileSize int64, imageFilename string) PodcastEpisode {
	imageUrl := ""
	if imageFilename != "" {
		imageUrl = hmnurl.BuildUserFile(imageFilename)
	}
	return PodcastEpisode{
		GUID:            episode.GUID.String(),
		Title:           episode.Title,
		Description:     episode.Description,
		DescriptionHtml: template.HTML(episode.DescriptionHtml),
		EpisodeNumber:   episode.EpisodeNumber,
		Url:             hmnurl.BuildPodcastEpisode(projectSlug, episode.GUID.String()),
		ImageUrl:        imageUrl,
		FileUrl:         hmnurl.BuildPodcastEpisodeFile(projectSlug, episode.AudioFile),
		FileSize:        audioFileSize,
		PublicationDate: episode.PublicationDate,
		Duration:        episode.Duration,
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

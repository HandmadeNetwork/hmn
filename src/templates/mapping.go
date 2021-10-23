package templates

import (
	"fmt"
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

		HasBlog:    p.BlogEnabled,
		HasForum:   p.ForumEnabled,
		HasLibrary: false, // TODO: port the library lol

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

		Name:       u.BestName(),
		Bio:        u.Bio,
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

// An online site/service for which we recognize the link
type LinkService struct {
	Name     string
	IconName string
	Regex    *regexp.Regexp
}

var LinkServices = []LinkService{
	{
		Name:     "YouTube",
		IconName: "youtube",
		Regex:    regexp.MustCompile(`youtube\.com/(c/)?(?P<userdata>[\w/-]+)$`),
	},
	{
		Name:     "Twitter",
		IconName: "twitter",
		Regex:    regexp.MustCompile(`twitter\.com/(?P<userdata>\w+)$`),
	},
	{
		Name:     "GitHub",
		IconName: "github",
		Regex:    regexp.MustCompile(`github\.com/(?P<userdata>[\w/-]+)$`),
	},
	{
		Name:     "Twitch",
		IconName: "twitch",
		Regex:    regexp.MustCompile(`twitch\.tv/(?P<userdata>[\w/-]+)$`),
	},
	{
		Name:     "Hitbox",
		IconName: "hitbox",
		Regex:    regexp.MustCompile(`hitbox\.tv/(?P<userdata>[\w/-]+)$`),
	},
	{
		Name:     "Patreon",
		IconName: "patreon",
		Regex:    regexp.MustCompile(`patreon\.com/(?P<userdata>[\w/-]+)$`),
	},
	{
		Name:     "SoundCloud",
		IconName: "soundcloud",
		Regex:    regexp.MustCompile(`soundcloud\.com/(?P<userdata>[\w/-]+)$`),
	},
	{
		Name:     "itch.io",
		IconName: "itch",
		Regex:    regexp.MustCompile(`(?P<userdata>[\w/-]+)\.itch\.io/?$`),
	},
}

func ParseKnownServicesForLink(link *models.Link) (service LinkService, userData string) {
	for _, svc := range LinkServices {
		match := svc.Regex.FindStringSubmatch(link.URL)
		if match != nil {
			return svc, match[svc.Regex.SubexpIndex("userdata")]
		}
	}
	return LinkService{}, ""
}

func LinkToTemplate(link *models.Link) Link {
	tlink := Link{
		Name:     link.Name,
		Url:      link.URL,
		LinkText: link.URL,
	}

	service, userData := ParseKnownServicesForLink(link)
	if tlink.Name == "" && service.Name != "" {
		tlink.Name = service.Name
	}
	if service.IconName != "" {
		tlink.Icon = service.IconName
	}
	if userData != "" {
		tlink.LinkText = userData
	}

	return tlink
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

		var mediaType TimelineItemMediaType
		var assetUrl string
		var thumbnailUrl string
		var width, height int
		if len(item.EmbedMedia) > 0 {
			mediaType = item.EmbedMedia[0].Type
			assetUrl = item.EmbedMedia[0].AssetUrl
			thumbnailUrl = item.EmbedMedia[0].ThumbnailUrl
			width = item.EmbedMedia[0].Width
			height = item.EmbedMedia[0].Height
		}

		builder.WriteString(`"media_type":`)
		builder.WriteString(strconv.Itoa(int(mediaType)))
		builder.WriteRune(',')

		builder.WriteString(`"width":`)
		builder.WriteString(strconv.Itoa(width))
		builder.WriteRune(',')

		builder.WriteString(`"height":`)
		builder.WriteString(strconv.Itoa(height))
		builder.WriteRune(',')

		builder.WriteString(`"asset_url":"`)
		builder.WriteString(assetUrl)
		builder.WriteString(`",`)

		builder.WriteString(`"thumbnail_url":"`)
		builder.WriteString(thumbnailUrl)
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

func DiscordUserToTemplate(d *models.DiscordUser) DiscordUser {
	var avatarUrl string // TODO: Default avatar image
	if d.Avatar != nil {
		avatarUrl = fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s.png", d.UserID, *d.Avatar)
	}

	return DiscordUser{
		Username:      d.Username,
		Discriminator: d.Discriminator,
		Avatar:        avatarUrl,
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

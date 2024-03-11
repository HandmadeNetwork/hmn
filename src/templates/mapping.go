package templates

import (
	"fmt"
	"html/template"
	"net/netip"
	"regexp"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/calendar"
	"git.handmade.network/hmn/hmn/src/hmndata"
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

func ProjectLogoUrl(p *models.Project, lightAsset *models.Asset, darkAsset *models.Asset, theme string) string {
	if theme == "dark" {
		if darkAsset != nil {
			return hmnurl.BuildS3Asset(darkAsset.S3Key)
		}
	} else {
		if lightAsset != nil {
			return hmnurl.BuildS3Asset(lightAsset.S3Key)
		}
	}
	return ""
}

func ProjectToTemplate(
	p *models.Project,
	url string,
) Project {
	return Project{
		ID:                p.ID,
		Name:              p.Name,
		Subdomain:         p.Subdomain(),
		Color1:            p.Color1,
		Color2:            p.Color2,
		Url:               url,
		Blurb:             p.Blurb,
		ParsedDescription: template.HTML(p.ParsedDescription),

		LifecycleBadgeClass: LifecycleBadgeClasses[p.Lifecycle],
		LifecycleString:     LifecycleBadgeStrings[p.Lifecycle],

		IsHMN: p.IsHMN(),

		HasBlog:  p.HasBlog(),
		HasForum: p.HasForums(),

		DateApproved: p.DateApproved,
	}
}

func ProjectAndStuffToTemplate(p *hmndata.ProjectAndStuff, url string, theme string) Project {
	res := ProjectToTemplate(&p.Project, url)
	res.Logo = ProjectLogoUrl(&p.Project, p.LogoLightAsset, p.LogoDarkAsset, theme)
	for _, o := range p.Owners {
		res.Owners = append(res.Owners, UserToTemplate(o, theme))
	}
	return res
}

var ProjectLifecycleValues = map[models.ProjectLifecycle]string{
	models.ProjectLifecycleActive: "active",
	models.ProjectLifecycleHiatus: "hiatus",
	models.ProjectLifecycleDead:   "dead",
	models.ProjectLifecycleLTS:    "done",
}

func ProjectLifecycleFromValue(value string) (models.ProjectLifecycle, bool) {
	for k, v := range ProjectLifecycleValues {
		if v == value {
			return k, true
		}
	}
	return models.ProjectLifecycleUnapproved, false
}

func ProjectToProjectSettings(
	p *models.Project,
	owners []*models.User,
	tag string,
	lightLogoUrl, darkLogoUrl string,
	currentTheme string,
) ProjectSettings {
	ownerUsers := make([]User, 0, len(owners))
	for _, owner := range owners {
		ownerUsers = append(ownerUsers, UserToTemplate(owner, currentTheme))
	}
	return ProjectSettings{
		Name:        p.Name,
		Slug:        p.Slug,
		Hidden:      p.Hidden,
		Featured:    p.Featured,
		Personal:    p.Personal,
		Lifecycle:   ProjectLifecycleValues[p.Lifecycle],
		Tag:         tag,
		Blurb:       p.Blurb,
		Description: p.Description,
		Owners:      ownerUsers,
		LightLogo:   lightLogoUrl,
		DarkLogo:    darkLogoUrl,
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

func UserAvatarDefaultUrl(currentTheme string) string {
	return hmnurl.BuildTheme("empty-avatar.svg", currentTheme, true)
}

func UserAvatarUrl(u *models.User, currentTheme string) string {
	if currentTheme == "" {
		currentTheme = "light"
	}
	avatar := ""
	if u != nil && u.AvatarAsset != nil {
		avatar = hmnurl.BuildS3Asset(u.AvatarAsset.S3Key)
	} else {
		avatar = UserAvatarDefaultUrl(currentTheme)
	}
	return avatar
}

func UserToTemplate(u *models.User, currentTheme string) User {
	if u == nil {
		return User{
			Name:      "Deleted user",
			AvatarUrl: UserAvatarUrl(nil, currentTheme),
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
		Status:   int(u.Status),

		Name:       u.BestName(),
		Bio:        u.Bio,
		Blurb:      u.Blurb,
		Signature:  u.Signature,
		DateJoined: u.DateJoined,
		ProfileUrl: hmnurl.BuildUserProfile(u.Username),
		AvatarUrl:  UserAvatarUrl(u, currentTheme),

		DarkTheme: u.DarkTheme,
		Timezone:  u.Timezone,

		DiscordSaveShowcase:                 u.DiscordSaveShowcase,
		DiscordDeleteSnippetOnMessageDelete: u.DiscordDeleteSnippetOnMessageDelete,

		IsEduTester: u.CanSeeUnpublishedEducationContent(),
		IsEduAuthor: u.CanAuthorEducation(),
	}
}

var UnknownUser = User{
	Name:      "Unknown User",
	AvatarUrl: UserAvatarUrl(nil, ""),
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
		jsonString = strings.ToValidUTF8(jsonString, "")
		jsonString = strings.ReplaceAll(jsonString, `\`, `\\`)
		jsonString = strings.ReplaceAll(jsonString, `"`, `\"`)
		jsonString = strings.ReplaceAll(jsonString, "\n", "\\n")
		jsonString = strings.ReplaceAll(jsonString, "\r", "\\r")
		jsonString = strings.ReplaceAll(jsonString, "\t", "\\t")
		jsonString = controlCharRegex.ReplaceAllString(jsonString, "")
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
		builder.WriteString(`",`)

		builder.WriteString(`"projects":[`)
		for j, proj := range item.Projects {
			if j > 0 {
				builder.WriteRune(',')
			}
			builder.WriteRune('{')

			builder.WriteString(`"name":"`)
			builder.WriteString(proj.Name)
			builder.WriteString(`",`)

			builder.WriteString(`"logo":"`)
			builder.WriteString(proj.Logo)
			builder.WriteString(`",`)

			builder.WriteString(`"url":"`)
			builder.WriteString(proj.Url)
			builder.WriteString(`"`)

			builder.WriteRune('}')
		}
		builder.WriteString(`]`)

		builder.WriteRune('}')
	}
	builder.WriteRune(']')
	return builder.String()
}

func SnippetEditProjectsToJSON(projects []Project) string {
	builder := strings.Builder{}
	builder.WriteRune('[')
	for i, proj := range projects {
		if i > 0 {
			builder.WriteRune(',')
		}
		builder.WriteRune('{')

		builder.WriteString(`"id":`)
		builder.WriteString(strconv.FormatInt(int64(proj.ID), 10))
		builder.WriteRune(',')

		builder.WriteString(`"name":"`)
		builder.WriteString(proj.Name)
		builder.WriteString(`",`)

		builder.WriteString(`"logo":"`)
		builder.WriteString(proj.Logo)
		builder.WriteRune('"')

		builder.WriteRune('}')
	}
	builder.WriteRune(']')
	return builder.String()
}

func PodcastToTemplate(podcast *models.Podcast, imageFilename string) Podcast {
	imageUrl := ""
	if imageFilename != "" {
		imageUrl = hmnurl.BuildUserFile(imageFilename)
	}
	return Podcast{
		Title:       podcast.Title,
		Description: podcast.Description,
		Language:    podcast.Language,
		ImageUrl:    imageUrl,
		Url:         hmnurl.BuildPodcast(),

		RSSUrl: hmnurl.BuildPodcastRSS(),
		// TODO(asaf): Move this to the db if we want to support user podcasts
		AppleUrl:   "https://podcasts.apple.com/us/podcast/the-handmade-network-podcast/id1507790631",
		GoogleUrl:  "https://www.google.com/podcasts?feed=aHR0cHM6Ly9oYW5kbWFkZS5uZXR3b3JrL3BvZGNhc3QvcG9kY2FzdC54bWw%3D",
		SpotifyUrl: "https://open.spotify.com/show/2Nd9NjXscrBbQwYULiYKiU",
	}
}

func PodcastEpisodeToTemplate(episode *models.PodcastEpisode, audioFileSize int64, imageFilename string) PodcastEpisode {
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
		Url:             hmnurl.BuildPodcastEpisode(episode.GUID.String()),
		ImageUrl:        imageUrl,
		FileUrl:         hmnurl.BuildPodcastEpisodeFile(episode.AudioFile),
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

func TagToTemplate(t *models.Tag) Tag {
	return Tag{
		Text: t.Text,
		// TODO: Url
	}
}

func EducationArticleToTemplate(a *models.EduArticle) EduArticle {
	res := EduArticle{
		Title:       a.Title,
		Slug:        a.Slug,
		Description: a.Description,
		Published:   a.Published,

		Url:       hmnurl.BuildEducationArticle(a.Slug),
		EditUrl:   hmnurl.BuildEducationArticleEdit(a.Slug),
		DeleteUrl: hmnurl.BuildEducationArticleDelete(a.Slug),

		Content: "NO CONTENT HERE FOLKS YOU DID A BUG",
	}
	switch a.Type {
	case models.EduArticleTypeArticle:
		res.Type = "article"
	case models.EduArticleTypeGlossary:
		res.Type = "glossary"
	}

	if a.CurrentVersion != nil {
		res.Content = template.HTML(a.CurrentVersion.ContentHTML)
	}

	return res
}

func CalendarEventToTemplate(ev *calendar.CalendarEvent) CalendarEvent {
	return CalendarEvent{
		Name:      ev.Name,
		Desc:      ev.Desc,
		StartTime: ev.StartTime.UTC(),
		EndTime:   ev.EndTime.UTC(),
		CalName:   ev.CalName,
	}
}

func maybeString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func maybeIp(ip *netip.Prefix) string {
	if ip == nil {
		return ""
	}

	return ip.String()
}

package templates

import (
	"fmt"
	"html/template"
	"net/netip"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/calendar"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/links"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/utils"
)

func PostToTemplate(p *models.Post, author *models.User) Post {
	return Post{
		ID: p.ID,

		// Urls not set here. They vary per thread type. Set 'em yourself!

		Preview:  p.Preview,
		ReadOnly: p.ReadOnly,

		Author: UserToTemplate(author),
		// No content. A lot of the time we don't have this handy and don't need it. See AddContentVersion.
		PostDate: p.PostDate,
	}
}

func (p *Post) AddContentVersion(ver models.PostVersion, editor *models.User) {
	p.Content = template.HTML(ver.TextParsed)
	p.IP = maybeIp(ver.IP)

	if editor != nil {
		editorTmpl := UserToTemplate(editor)
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

// TODO(redesign): Remove one or the other of these from the database entirely.
func ProjectLogoUrl(p *models.Project, lightAsset *models.Asset, darkAsset *models.Asset) string {
	if lightAsset != nil {
		return hmnurl.BuildS3Asset(lightAsset.S3Key)
	}
	if darkAsset != nil {
		return hmnurl.BuildS3Asset(darkAsset.S3Key)
	}
	return ""
}

func ProjectToTemplate(
	p *models.Project,
) Project {
	return Project{
		ID:                p.ID,
		Name:              p.Name,
		Subdomain:         p.Subdomain(),
		Color1:            p.Color1,
		Color2:            p.Color2,
		Url:               hmndata.UrlContextForProject(p).BuildHomepage(),
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

func ProjectAndStuffToTemplate(p *hmndata.ProjectAndStuff) Project {
	res := ProjectToTemplate(&p.Project)
	res.Logo = ProjectLogoUrl(&p.Project, p.LogoLightAsset, p.LogoDarkAsset)
	for _, o := range p.Owners {
		res.Owners = append(res.Owners, UserToTemplate(o))
	}
	if p.HeaderImage != nil {
		res.HeaderImage = hmnurl.BuildS3Asset(p.HeaderImage.S3Key)
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
	lightLogo, darkLogo, headerImage *models.Asset,
) ProjectSettings {
	ownerUsers := make([]User, 0, len(owners))
	for _, owner := range owners {
		ownerUsers = append(ownerUsers, UserToTemplate(owner))
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
		LightLogo:   AssetToTemplate(lightLogo),
		DarkLogo:    AssetToTemplate(darkLogo),
		HeaderImage: AssetToTemplate(headerImage),
	}
}

func AssetToTemplate(a *models.Asset) *Asset {
	if a == nil {
		return nil
	}

	return &Asset{
		Url: hmnurl.BuildS3Asset(a.S3Key),

		ID:       a.ID.String(),
		Filename: a.Filename,
		Size:     a.Size,
		MimeType: a.MimeType,
		Width:    a.Width,
		Height:   a.Height,
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

func UserAvatarDefaultUrl(theme string) string {
	// TODO(redesign): Get rid of theme here
	return hmnurl.BuildTheme("empty-avatar.svg", theme, true)
}

func UserAvatarUrl(u *models.User) string {
	avatar := UserAvatarDefaultUrl("light")
	if u != nil && u.AvatarAsset != nil {
		avatar = hmnurl.BuildS3Asset(u.AvatarAsset.S3Key)
	}
	return avatar
}

func UserToTemplate(u *models.User) User {
	if u == nil {
		return User{
			Name:      "Deleted user",
			AvatarUrl: UserAvatarUrl(nil),
		}
	}

	email := ""
	if u.ShowEmail {
		// TODO(asaf): Always show email to admins
		email = u.Email
	}

	var discordUser *DiscordUser
	if u.DiscordUser != nil {
		du := DiscordUserToTemplate(u.DiscordUser)
		discordUser = &du
	}

	return User{
		ID:       u.ID,
		Username: u.Username,
		Email:    email,
		IsStaff:  u.IsStaff,
		Status:   int(u.Status),
		Featured: u.Featured,

		Name:       u.BestName(),
		Bio:        u.Bio,
		Blurb:      u.Blurb,
		Signature:  u.Signature,
		DateJoined: u.DateJoined,
		ProfileUrl: hmnurl.BuildUserProfile(u.Username),
		Avatar:     AssetToTemplate(u.AvatarAsset),
		AvatarUrl:  UserAvatarUrl(u),

		Timezone: u.Timezone,

		DiscordSaveShowcase:                 u.DiscordSaveShowcase,
		DiscordDeleteSnippetOnMessageDelete: u.DiscordDeleteSnippetOnMessageDelete,

		IsEduTester: u.CanSeeUnpublishedEducationContent(),
		IsEduAuthor: u.CanAuthorEducation(),

		DiscordUser: discordUser,
	}
}

var UnknownUser = User{
	Name:      "Unknown User",
	AvatarUrl: UserAvatarUrl(nil),
}

func LinkToTemplate(link *models.Link) Link {
	service, username := links.ParseKnownServicesForUrl(link.URL)
	return Link{
		Name:        link.Name,
		Url:         link.URL,
		ServiceName: service.Name,
		Icon:        service.IconName,
		Username:    username,
		Primary:     link.Primary,
	}
}

func LinksToTemplate(links []*models.Link) []Link {
	res := make([]Link, len(links))
	for i, link := range links {
		res[i] = LinkToTemplate(link)
	}
	return res
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

		// TODO(redesign): This only serializes a single piece of media.
		var mediaType TimelineItemMediaType
		var assetUrl string
		var thumbnailUrl string
		var width, height int
		if len(item.Media) > 0 {
			mediaType = item.Media[0].Type
			assetUrl = item.Media[0].AssetUrl
			thumbnailUrl = item.Media[0].ThumbnailUrl
			width = item.Media[0].Width
			height = item.Media[0].Height
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

func JamToBannerEvent(jam hmndata.Jam) BannerEvent {
	return BannerEvent{
		Slug:           jam.Slug,
		DaysUntilStart: utils.DaysUntil(jam.StartTime),
		DaysUntilEnd:   utils.DaysUntil(jam.EndTime),
		StartTimeUnix:  jam.StartTime.Unix(),
		EndTimeUnix:    jam.EndTime.Unix(),
		Url:            hmnurl.BuildJamIndexAny(jam.UrlSlug),
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

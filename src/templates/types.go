package templates

import (
	"html/template"
	"time"
)

type BaseData struct {
	Title           string
	CanonicalLink   string
	OpenGraphItems  []OpenGraphItem
	BackgroundImage BackgroundImage
	Theme           string
	BodyClasses     []string
	Breadcrumbs     []Breadcrumb

	CurrentUrl    string
	LoginPageUrl  string
	ProjectCSSUrl string

	Project Project
	User    *User
	Session *Session

	IsProjectPage  bool
	Header         Header
	Footer         Footer
	MathjaxEnabled bool
}

type Header struct {
	AdminUrl           string
	UserSettingsUrl    string
	LoginActionUrl     string
	LogoutActionUrl    string
	RegisterUrl        string
	HMNHomepageUrl     string
	ProjectHomepageUrl string
	ProjectIndexUrl    string
	BlogUrl            string
	ForumsUrl          string
	WikiUrl            string
	LibraryUrl         string
	ManifestoUrl       string
	EpisodeGuideUrl    string
	EditUrl            string
	SearchActionUrl    string
}

type Footer struct {
	HomepageUrl                string
	AboutUrl                   string
	ManifestoUrl               string
	CodeOfConductUrl           string
	CommunicationGuidelinesUrl string
	ProjectIndexUrl            string
	ForumsUrl                  string
	ContactUrl                 string
	SitemapUrl                 string
}

type Thread struct {
	Title string

	Locked bool
	Sticky bool
}

type Post struct {
	ID int

	Url       string
	DeleteUrl string
	EditUrl   string
	ReplyUrl  string
	QuoteUrl  string

	Preview  string
	ReadOnly bool

	Author   *User
	Content  template.HTML
	PostDate time.Time

	Editor     *User
	EditDate   time.Time
	EditReason string

	IP string
}

type Project struct {
	Name              string
	Subdomain         string
	Color1            string
	Color2            string
	Url               string
	Blurb             string
	ParsedDescription template.HTML
	Owners            []User

	Logo string

	LifecycleBadgeClass string
	LifecycleString     string

	IsHMN bool

	HasBlog    bool
	HasForum   bool
	HasWiki    bool
	HasLibrary bool

	UUID         string
	DateApproved time.Time
}

type User struct {
	ID          int
	Username    string
	Email       string
	IsSuperuser bool
	IsStaff     bool

	Name       string
	Blurb      string
	Bio        string
	Signature  string
	DateJoined time.Time
	AvatarUrl  string
	ProfileUrl string

	DarkTheme     bool
	Timezone      string
	ProfileColor1 string
	ProfileColor2 string

	CanEditLibrary                      bool
	DiscordSaveShowcase                 bool
	DiscordDeleteSnippetOnMessageDelete bool
}

type Link struct {
	Key  string
	Name string
	Url  string
	Icon string
}

type Podcast struct {
	Title       string
	Description string
	Language    string
	ImageUrl    string
	Url         string

	RSSUrl     string
	AppleUrl   string
	GoogleUrl  string
	SpotifyUrl string
}

type PodcastEpisode struct {
	GUID            string
	Title           string
	Description     string
	DescriptionHtml template.HTML
	EpisodeNumber   int
	Url             string
	ImageUrl        string
	FileUrl         string
	FileSize        int64
	PublicationDate time.Time
	Duration        int
}

type Notice struct {
	Content template.HTML
	Class   string
}

type Session struct {
	CSRFToken string
}

type OpenGraphItem struct {
	Property string
	Name     string
	Value    string
}

type BackgroundImage struct {
	Url  string
	Size string // A valid CSS background-size value
}

type PostType int

const (
	PostTypeUnknown PostType = iota
	PostTypeBlogPost
	PostTypeBlogComment
	PostTypeForumThread
	PostTypeForumReply
	PostTypeWikiCreate
	PostTypeWikiTalk
	PostTypeWikiEdit
	PostTypeLibraryComment
)

// Data from post_list_item.html
type PostListItem struct {
	Title       string
	Url         string
	UUID        string
	Breadcrumbs []Breadcrumb

	PostType       PostType
	PostTypePrefix string

	User User
	Date time.Time

	Unread       bool
	Classes      string
	Preview      string
	LastEditDate time.Time
}

// Data from thread_list_item.html
type ThreadListItem struct {
	Title string
	Url   string

	FirstUser User
	FirstDate time.Time
	LastUser  User
	LastDate  time.Time

	Unread  bool
	Classes string
	Content string
}

type TimelineType int

const (
	TimelineTypeUnknown TimelineType = iota

	TimelineTypeForumThread
	TimelineTypeForumReply

	TimelineTypeBlogPost
	TimelineTypeBlogComment

	TimelineTypeWikiCreate
	TimelineTypeWikiEdit
	TimelineTypeWikiTalk

	TimelineTypeLibraryComment

	TimelineTypeSnippetImage
	TimelineTypeSnippetVideo
	TimelineTypeSnippetAudio
	TimelineTypeSnippetYoutube
)

type TimelineItem struct {
	Type      TimelineType
	TypeTitle string
	Class     string
	Date      time.Time
	Url       string
	UUID      string

	OwnerAvatarUrl string
	OwnerName      string
	OwnerUrl       string
	Description    template.HTML

	DiscordMessageUrl string
	Width             int
	Height            int
	AssetUrl          string
	MimeType          string
	YoutubeID         string

	Title       string
	Breadcrumbs []Breadcrumb
}

type ProjectCardData struct {
	Project *Project
	Classes string
}

type Breadcrumb struct {
	Name, Url string
	Current   bool
}

type Pagination struct {
	Current int
	Total   int

	FirstUrl    string
	LastUrl     string
	PreviousUrl string
	NextUrl     string
}

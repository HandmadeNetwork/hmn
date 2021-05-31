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

	LoginPageUrl  string
	ProjectCSSUrl string

	Project Project
	User    *User

	Header Header
	Footer Footer
}

type Header struct {
	AdminUrl           string
	MemberSettingsUrl  string
	LoginActionUrl     string
	LogoutActionUrl    string
	RegisterUrl        string
	HMNHomepageUrl     string
	ProjectHomepageUrl string
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
	EditIP     string
	EditReason string

	IP string
}

type Project struct {
	Name      string
	Subdomain string
	Color1    string
	Color2    string
	Url       string
	Blurb     string
	Owners    []User

	LogoDark  string
	LogoLight string

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
	Title       string
	Url         string
	Breadcrumbs []Breadcrumb

	FirstUser User
	FirstDate time.Time
	LastUser  User
	LastDate  time.Time

	Unread  bool
	Classes string
	Content string
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

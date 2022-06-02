package templates

import (
	"html/template"
	"time"
)

type BaseData struct {
	Title            string
	CanonicalLink    string
	OpenGraphItems   []OpenGraphItem
	BackgroundImage  BackgroundImage
	Theme            string
	BodyClasses      []string
	Breadcrumbs      []Breadcrumb
	Notices          []Notice
	ReportIssueEmail string

	CurrentUrl        string
	CurrentProjectUrl string
	LoginPageUrl      string
	ProjectCSSUrl     string

	Project Project
	User    *User
	Session *Session

	IsProjectPage bool
	Header        Header
	Footer        Footer
}

func (bd *BaseData) AddImmediateNotice(class, content string) {
	bd.Notices = append(bd.Notices, Notice{
		Class:   class,
		Content: template.HTML(content),
	})
}

type Header struct {
	AdminUrl          string
	UserProfileUrl    string
	UserSettingsUrl   string
	LoginActionUrl    string
	LogoutActionUrl   string
	ForgotPasswordUrl string
	RegisterUrl       string

	HMNHomepageUrl  string
	ProjectIndexUrl string
	PodcastUrl      string
	ForumsUrl       string
	LibraryUrl      string

	Project *ProjectHeader
}

type ProjectHeader struct {
	HasForums       bool
	HasBlog         bool
	HasEpisodeGuide bool
	CanEdit         bool
	ForumsUrl       string
	BlogUrl         string
	EpisodeGuideUrl string
	EditUrl         string
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
	SearchActionUrl            string
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

	Preview  string
	ReadOnly bool

	Author   User
	Content  template.HTML
	PostDate time.Time

	AuthorNumPosts    int
	AuthorNumProjects int

	Editor     *User
	EditDate   time.Time
	EditReason string

	IP string

	ReplyPost *Post
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

	HasBlog  bool
	HasForum bool

	UUID         string
	DateApproved time.Time
}

type ProjectSettings struct {
	Name      string
	Slug      string
	Hidden    bool
	Featured  bool
	Personal  bool
	Lifecycle string
	Tag       string

	Blurb       string
	Description string
	LinksText   string
	Owners      []User

	LightLogo string
	DarkLogo  string
}

type User struct {
	ID       int
	Username string
	Email    string
	IsStaff  bool
	Status   int

	Name       string
	Blurb      string
	Bio        string
	Signature  string
	DateJoined time.Time
	AvatarUrl  string
	ProfileUrl string

	DarkTheme bool
	ShowEmail bool
	Timezone  string

	CanEditLibrary                      bool
	DiscordSaveShowcase                 bool
	DiscordDeleteSnippetOnMessageDelete bool
}

type Link struct {
	Name     string
	Url      string
	LinkText string
	Icon     string
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

// NOTE(asaf): See /src/rawdata/scss/_notices.scss for a list of classes.
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

type TimelineItem struct {
	Date              time.Time
	Title             string
	TypeTitle         string
	FilterTitle       string
	Breadcrumbs       []Breadcrumb
	Url               string
	DiscordMessageUrl string

	OwnerAvatarUrl string
	OwnerName      string
	OwnerUrl       string

	Tags        []Tag
	Description template.HTML

	PreviewMedia TimelineItemMedia
	EmbedMedia   []TimelineItemMedia

	SmallInfo           bool
	AllowTitleWrap      bool
	TruncateDescription bool
	CanShowcase         bool // whether this snippet can be shown in a showcase gallery
}

type TimelineItemMediaType int

const (
	TimelineItemMediaTypeUnknown TimelineItemMediaType = iota
	TimelineItemMediaTypeImage
	TimelineItemMediaTypeVideo
	TimelineItemMediaTypeAudio
	TimelineItemMediaTypeEmbed
)

type TimelineItemMedia struct {
	Type                TimelineItemMediaType
	AssetUrl            string
	EmbedHTML           template.HTML
	ThumbnailUrl        string
	MimeType            string
	Width, Height       int
	Filename            string
	FileSize            int
	ExtraOpenGraphItems []OpenGraphItem
}

type ProjectCardData struct {
	Project *Project
	Classes string
}

type ImageSelectorData struct {
	Name     string
	Src      string
	Required bool
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

type EmailBaseData struct {
	To        template.HTML
	From      template.HTML
	Subject   template.HTML
	Separator template.HTML
}

type DiscordUser struct {
	Username      string
	Discriminator string
	Avatar        string
}

type Tag struct {
	Text string
	Url  string
}

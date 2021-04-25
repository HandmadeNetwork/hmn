package templates

import "time"

type BaseData struct {
	Title           string
	CanonicalLink   string
	OpenGraphItems  []OpenGraphItem
	BackgroundImage BackgroundImage
	Theme           string
	BodyClasses     []string

	Project Project
	User    *User
}

type Thread struct {
	Title string

	Locked    bool
	Sticky    bool
	Moderated bool
}

type Post struct {
	Author   User
	Preview  string
	ReadOnly bool

	Content string

	IP string
}

type Project struct {
	Name      string
	Subdomain string
	Color1    string
	Color2    string

	IsHMN bool

	HasBlog    bool
	HasForum   bool
	HasWiki    bool
	HasLibrary bool
}

type User struct {
	Username    string
	Email       string
	IsSuperuser bool
	IsStaff     bool

	Name       string
	Blurb      string
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

// Data from post_list_item.html
type PostListItem struct {
	Title       string
	Url         string
	Breadcrumbs []Breadcrumb
	User        User
	Date        time.Time
	Unread      bool
	Classes     string
	Content     string
}

type Breadcrumb struct {
	Name, Url string
}

type Pagination struct {
	Current int
	Total   int

	FirstUrl    string
	LastUrl     string
	PreviousUrl string
	NextUrl     string
}

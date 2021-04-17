package templates

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

type Post struct {
	Preview  string
	ReadOnly bool

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

	Name      string
	Blurb     string
	Signature string
	// TODO: Avatar??

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

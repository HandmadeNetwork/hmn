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

type Project struct {
	Name      string
	Subdomain string
	Color     string

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

package templates

type BaseData struct {
	Title           string
	CanonicalLink   string
	OpenGraphItems  []OpenGraphItem
	BackgroundImage BackgroundImage
	ProjectColor    string
	Theme           string
	BodyClasses     []string
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

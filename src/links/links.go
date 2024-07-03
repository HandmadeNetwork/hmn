package links

import "regexp"

//
// This is all in its own package so we can compile it to wasm without building extra junk.
//

// An online site/service for which we recognize the link
type Service struct {
	Name     string
	IconName string
	Regex    *regexp.Regexp
}

var Services = []Service{
	// {
	// 	Name:     "itch.io",
	// 	IconName: "itch",
	// 	Regex:    regexp.MustCompile(`://(?P<username>[\w-]+)\.itch\.io`),
	// },
	{
		Name:     "App Store",
		IconName: "app-store",
		Regex:    regexp.MustCompile(`^https?://apps.apple.com`),
	},
	{
		Name:     "Bluesky",
		IconName: "bluesky",
		Regex:    regexp.MustCompile(`^https?://bsky.app/profile/(?P<username>[\w.-]+)$`),
	},
	{
		Name:     "Discord",
		IconName: "discord",
		Regex:    regexp.MustCompile(`^https?://discord\.gg`),
	},
	{
		Name:     "GitHub",
		IconName: "github",
		Regex:    regexp.MustCompile(`^https?://github\.com/(?P<username>[\w/-]+)`),
	},
	{
		Name:     "GitLab",
		IconName: "gitlab",
		Regex:    regexp.MustCompile(`^https?://gitlab\.com/(?P<username>[\w/-]+)`),
	},
	{
		Name:     "Google Play",
		IconName: "google-play",
		Regex:    regexp.MustCompile(`^https?://play\.google\.com`),
	},
	{
		Name:     "Patreon",
		IconName: "patreon",
		Regex:    regexp.MustCompile(`^https?://patreon\.com/(?P<username>[\w-]+)`),
	},
	{
		Name:     "Twitch",
		IconName: "twitch",
		Regex:    regexp.MustCompile(`^https?://twitch\.tv/(?P<username>[\w/-]+)`),
	},
	{
		Name:     "Twitter",
		IconName: "twitter",
		Regex:    regexp.MustCompile(`^https?://(twitter|x)\.com/(?P<username>\w+)`),
	},
	{
		Name:     "Vimeo",
		IconName: "vimeo",
		Regex:    regexp.MustCompile(`^https?://vimeo\.com/(?P<username>\w+)`),
	},
	{
		Name:     "YouTube",
		IconName: "youtube",
		Regex:    regexp.MustCompile(`youtube\.com/(c/)?(?P<username>[@\w/-]+)$`),
	},
}

func ParseKnownServicesForUrl(url string) (service Service, username string) {
	for _, svc := range Services {
		match := svc.Regex.FindStringSubmatch(url)
		if match != nil {
			username := ""
			if idx := svc.Regex.SubexpIndex("username"); idx >= 0 {
				username = match[idx]
			}

			return svc, username
		}
	}
	return Service{
		IconName: "website",
	}, ""
}

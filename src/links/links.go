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
		Name:     "Codeberg",
		IconName: "codeberg",
		Regex:    regexp.MustCompile(`^https?://codeberg\.org/(?P<username>[w/-]+)?`),
	},
	{
		Name:     "Discord",
		IconName: "discord",
		Regex:    regexp.MustCompile(`^https?://discord\.(gg|com)`),
	},
	{
		Name:     "GitHub",
		IconName: "github",
		Regex:    regexp.MustCompile(`^https?://(\w+\.)?github\.com/(?P<username>[w/-]+)?`),
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
		Name:     "Mastodon",
		IconName: "mastodon",
		Regex:    regexp.MustCompile(`^https?://[a-zA-Z0-9_\.-]*\b(mas\.?to|mstdn)[a-zA-Z0-9_\.-]*/(?P<username>@\w+)?`),
	},
	{
		Name:     "Patreon",
		IconName: "patreon",
		Regex:    regexp.MustCompile(`^https?://(www\.)?patreon\.com/(?P<username>[\w-]+)`),
	},
	{
		Name:     "Steam",
		IconName: "steam",
		Regex:    regexp.MustCompile(`^https?://store\.steampowered\.com/`),
	},
	{
		Name:     "Trello",
		IconName: "trello",
		Regex:    regexp.MustCompile(`^https?://trello\.com/b/`),
	},
	{
		Name:     "Twitch",
		IconName: "twitch",
		Regex:    regexp.MustCompile(`^https?://twitch\.tv/(?P<username>[\w/-]+)`),
	},
	{
		Name:     "X",
		IconName: "twitter-x", // NOTE(ben): Still the worst rebrand of all time
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
		Regex:    regexp.MustCompile(`^https?://((www\.)?youtube\.com/(watch\?|(c/)?(?P<username>[@w/-]+))?|youtu\.be/)`),
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

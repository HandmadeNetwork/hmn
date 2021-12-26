package website

import (
	"fmt"
	"strings"

	"git.handmade.network/hmn/hmn/src/models"
)

type ParsedLink struct {
	Name string
	Url  string
}

func ParseLinks(text string) []ParsedLink {
	lines := strings.Split(text, "\n")
	res := make([]ParsedLink, 0, len(lines))
	for _, line := range lines {
		linkParts := strings.SplitN(line, " ", 2)
		url := strings.TrimSpace(linkParts[0])
		name := ""
		if len(linkParts) > 1 {
			name = strings.TrimSpace(linkParts[1])
		}
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			continue
		}
		res = append(res, ParsedLink{Name: name, Url: url})
	}
	return res
}

func LinksToText(links []interface{}) string {
	linksText := ""
	for _, l := range links {
		link := l.(*models.Link)
		linksText += fmt.Sprintf("%s %s\n", link.URL, link.Name)
	}
	return linksText
}

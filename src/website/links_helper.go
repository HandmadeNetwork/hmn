package website

import (
	"encoding/json"
	"fmt"
	"strings"

	"git.handmade.network/hmn/hmn/src/models"
)

type ParsedLink struct {
	Name    string `json:"name"`
	Url     string `json:"url"`
	Primary bool   `json:"primary"`
}

func ParseLinks(text string) []ParsedLink {
	var links []ParsedLink
	err := json.Unmarshal([]byte(text), &links)
	if err != nil {
		return nil
	}
	return links
}

// TODO: Clean up use in user profiles I guess
func LinksToText(links []*models.Link) string {
	var linksText strings.Builder
	for _, link := range links {
		linksText.WriteString(fmt.Sprintf("%s %s\n", link.URL, link.Name))
	}
	return linksText.String()
}

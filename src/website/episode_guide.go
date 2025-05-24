package website

import (
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/templates"
)

func CineraIndex(c *RequestContext) ResponseData {
	topic := c.PathParams["topic"]
	slug := c.CurrentProject.Slug

	_, foundTopic := topicsForProject(slug, topic)
	if foundTopic == "" {
		return FourOhFour(c)
	}

	indexPath := path.Join(config.Config.EpisodeGuide.CineraOutputPath, slug, foundTopic, fmt.Sprintf("%s.index", foundTopic))

	content, err := os.ReadFile(indexPath)
	if err != nil {
		return FourOhFour(c)
	}

	var res ResponseData
	res.Write(content)

	return res
}

var episodeListRegex = regexp.MustCompile(`(?ism)^.*<body>(?P<guide>.*)</body>.*$`)

type EpisodeListData struct {
	templates.BaseData
	Content      template.HTML
	CurrentTopic string
	Topics       []templates.Link
}

func EpisodeList(c *RequestContext) ResponseData {
	topic := c.PathParams["topic"]
	slug := c.CurrentProject.Slug

	defaultTopic, hasEpisodeGuide := config.Config.EpisodeGuide.Projects[slug]

	if !hasEpisodeGuide {
		return c.Redirect(c.UrlContext.BuildHomepage(), http.StatusSeeOther)
	}

	if slug == "hero" {
		// NOTE(asaf): Manual override for HMH
		return c.Redirect(fmt.Sprintf("https://guide.handmadehero.org/%s", topic), http.StatusSeeOther)
	}

	if topic == "" {
		return c.Redirect(c.UrlContext.BuildEpisodeList(defaultTopic), http.StatusSeeOther)
	}

	allTopics, foundTopic := topicsForProject(slug, topic)
	if foundTopic == "" {
		return FourOhFour(c)
	}

	htmlPath := path.Join(config.Config.EpisodeGuide.CineraOutputPath, slug, foundTopic, "index.html")

	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		return FourOhFour(c)
	}
	matches := episodeListRegex.FindStringSubmatch(string(htmlContent))
	if matches == nil || matches[episodeListRegex.SubexpIndex("guide")] == "" {
		c.Logger.Error().Str("Filename", htmlPath).Msg("Episode guide index.html can't be parsed.")
		return FourOhFour(c)
	}
	guide := matches[episodeListRegex.SubexpIndex("guide")]

	var topicLinks []templates.Link
	for _, t := range allTopics {
		url := ""
		if t != foundTopic {
			url = c.UrlContext.BuildEpisodeList(t)
		}
		topicLinks = append(topicLinks, templates.Link{Username: t, Url: url})
	}

	var res ResponseData
	baseData := getBaseData(c, "Episode Guide", nil)
	res.MustWriteTemplate("episode_list.html", EpisodeListData{
		BaseData:     baseData,
		Content:      template.HTML(guide),
		CurrentTopic: foundTopic,
		Topics:       topicLinks,
	}, c.Perf)
	return res
}

var episodeTitleRegex = regexp.MustCompile(`(?ism)<span class="episode_name">(?P<title>.*?)</span>`)
var episodeContentRegex = regexp.MustCompile(`(?ism)<body>(?P<content>.*)</body>`)

type EpisodeData struct {
	templates.BaseData
	Content template.HTML
}

func Episode(c *RequestContext) ResponseData {
	episode := c.PathParams["episode"]
	topic := c.PathParams["topic"]
	slug := c.CurrentProject.Slug

	_, hasEpisodeGuide := config.Config.EpisodeGuide.Projects[slug]

	if !hasEpisodeGuide {
		return c.Redirect(c.UrlContext.BuildHomepage(), http.StatusSeeOther)
	}

	if slug == "hero" {
		// NOTE(asaf): Manual override for HMH
		return c.Redirect(fmt.Sprintf("https://guide.handmadehero.org/%s/%s", topic, episode), http.StatusSeeOther)
	}

	_, foundTopic := topicsForProject(slug, topic)
	if foundTopic == "" {
		return FourOhFour(c)
	}

	foundEpisode := findEpisode(slug, foundTopic, episode)
	if foundEpisode == "" {
		return FourOhFour(c)
	}

	htmlPath := path.Join(config.Config.EpisodeGuide.CineraOutputPath, slug, foundTopic, "episode", foundTopic, foundEpisode, "index.html")
	htmlContent, err := os.ReadFile(htmlPath)
	if err != nil {
		return FourOhFour(c)
	}

	titleMatches := episodeTitleRegex.FindStringSubmatch(string(htmlContent))
	title := fmt.Sprintf("%s Episode Guide", c.CurrentProject.Name)
	if titleMatches != nil && titleMatches[episodeTitleRegex.SubexpIndex("title")] != "" {
		title = fmt.Sprintf("%s | %s", titleMatches[episodeTitleRegex.SubexpIndex("title")], title)
	}

	contentMatches := episodeContentRegex.FindStringSubmatch(string(htmlContent))
	if contentMatches == nil || contentMatches[episodeContentRegex.SubexpIndex("content")] == "" {
		c.Logger.Error().Str("filename", htmlPath).Msg("Episode file can't be parsed.")
		return FourOhFour(c)
	}
	content := contentMatches[episodeContentRegex.SubexpIndex("content")]

	var res ResponseData
	baseData := getBaseData(c, title, nil)
	res.MustWriteTemplate("episode.html", EpisodeData{
		BaseData: baseData,
		Content:  template.HTML(content),
	}, c.Perf)
	return res
}

func topicsForProject(projectSlug string, requestedTopic string) ([]string, string) {
	searchPath := path.Join(config.Config.EpisodeGuide.CineraOutputPath, projectSlug)
	entries, err := fs.ReadDir(os.DirFS(searchPath), ".")
	if err != nil {
		return nil, ""
	}
	var allTopics []string
	foundTopic := ""
	for _, entry := range entries {
		if entry.IsDir() {
			t := entry.Name()
			allTopics = append(allTopics, t)
			if strings.ToLower(t) == strings.ToLower(requestedTopic) {
				foundTopic = t
			}
		}
	}

	return allTopics, foundTopic
}

// NOTE(asaf): Assuming topic is valid. Please verify before calling.
func findEpisode(projectSlug string, topic string, requestedEpisode string) string {
	searchPath := path.Join(config.Config.EpisodeGuide.CineraOutputPath, projectSlug, topic, "episode", topic) // NOTE(asaf): Yes. We have `topic` twice in the path.
	entries, err := fs.ReadDir(os.DirFS(searchPath), ".")
	if err != nil {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			episode := entry.Name()
			if strings.ToLower(episode) == strings.ToLower(requestedEpisode) {
				return episode
			}
		}
	}

	return ""
}

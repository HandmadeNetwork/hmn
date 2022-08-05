package embed

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/utils"
)

var DownloadTooBigError = errors.New("download too big")
var NoEmbedFound = errors.New("no embed found")

type Embeddable struct {
	Url  string
	File *Embed
}

type Embed struct {
	Data        []byte
	ContentType string
	Filename    string
}

var EmbeddableUrlRegex = regexp.MustCompile(`^https?://(youtu\.be|(www\.)?youtube\.com/watch)`)

func IsUrlEmbeddable(u string) bool {
	return EmbeddableUrlRegex.MatchString(u)
}

func GetEmbeddableFromUrls(ctx context.Context, urls []string, maxSize int, httpTimeout time.Duration, httpMaxAttempts int) (*Embeddable, error) {
	embedError := NoEmbedFound
	for _, urlStr := range urls {
		u, err := url.Parse(urlStr)
		if err != nil {
			continue
		}
		if u.Scheme == "" {
			u.Scheme = "https"
			urlStr = u.String()
		}

		if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			continue
		}

		if IsUrlEmbeddable(urlStr) {
			result := Embeddable{
				Url: urlStr,
			}
			return &result, nil
		}

		if httpMaxAttempts > 0 {
			httpMaxAttempts -= 1
			embed, err := FetchEmbed(ctx, urlStr, httpTimeout, maxSize)
			if err != nil {
				embedError = err
				continue
			}
			result := Embeddable{
				File: embed,
			}
			return &result, nil
		}
	}
	return nil, embedError
}

// If the url points to a file, downloads and returns the file.
// If the url points to an html page, parses opengraph and tries to fetch an image/video/audio file according to that.
// maxSize only limits the actual filesize. In the case of html we always fetch up to 100kb even if maxSize is smaller.
func FetchEmbed(ctx context.Context, urlStr string, timeout time.Duration, maxSize int) (*Embed, error) {
	logging.ExtractLogger(ctx).Debug().Msg("Fetching embed")
	client := &http.Client{
		Timeout: timeout,
	}
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, err
	}
	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if res.StatusCode < 200 || res.StatusCode > 299 {
		return nil, NoEmbedFound
	}
	contentType := res.Header.Get("Content-Type")
	logging.ExtractLogger(ctx).Debug().Str("type", contentType).Msg("Got first result")
	if strings.HasPrefix(contentType, "text/html") || strings.HasPrefix(contentType, "application/html") {
		var buffer bytes.Buffer
		_, err := io.CopyN(&buffer, res.Body, 100*1024) // NOTE(asaf): If the opengraph stuff isn't in the first 100kb, we don't care.
		res.Body.Close()
		if err != nil && !errors.Is(err, io.EOF) {
			return nil, err
		}
		partialHtml := buffer.Bytes()
		urlStr = ExtractEmbedFromOpenGraph(partialHtml)
		logging.ExtractLogger(ctx).Debug().Str("url", urlStr).Msg("Got ograph")
		if urlStr == "" {
			return nil, NoEmbedFound
		}

		req, err = http.NewRequestWithContext(ctx, "GET", urlStr, nil)
		if err != nil {
			return nil, err
		}
		res, err = client.Do(req)
		if err != nil {
			return nil, err
		}
		if res.StatusCode < 200 || res.StatusCode > 299 {
			return nil, NoEmbedFound
		}
		contentType = res.Header.Get("Content-Type")
	}

	var buffer bytes.Buffer
	n, err := io.CopyN(&buffer, res.Body, int64(maxSize+1))
	res.Body.Close()
	if err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	filename := ""
	u, err := url.Parse(urlStr)
	if err == nil {
		lastSlash := utils.IntMax(strings.LastIndex(u.Path, "/"), 0)
		filename = u.Path[lastSlash:]
	}
	result := Embed{
		Data:        buffer.Bytes(),
		ContentType: contentType,
		Filename:    filename,
	}
	if n == int64(maxSize+1) {
		err = DownloadTooBigError
	} else {
		err = nil
	}
	return &result, err
}

var metaRegex = regexp.MustCompile(`<meta\s+([^>]+)/?>`)
var metaAttrRegex = regexp.MustCompile(`(?P<key>\w+)="(?P<value>[^"]+)"`)

var OGKeys = []string{
	"og:audio",
	"og:video",
	"og:image",
	"og:audio:url",
	"og:image:url",
	"og:video:url",
	"og:audio:secure_url",
	"og:image:secure_url",
	"og:video:secure_url",
	"twitter:image",
}

// Tries to find an opengraph image/video/audio url in the provided html
// Since we only need to look at meta tags in the head, we don't need the full html document.
func ExtractEmbedFromOpenGraph(partialHtml []byte) string {
	keyIdx := metaAttrRegex.SubexpIndex("key")
	valueIdx := metaAttrRegex.SubexpIndex("value")
	html := string(partialHtml)
	matches := metaRegex.FindAllStringSubmatch(html, -1)
	for _, m := range matches {
		if len(m) > 1 {
			content := ""
			prop := ""
			attrs := metaAttrRegex.FindAllStringSubmatch(m[1], -1)
			for _, attr := range attrs {
				key := attr[keyIdx]
				value := attr[valueIdx]
				if key == "name" || key == "property" {
					for _, ogKey := range OGKeys {
						if value == ogKey {
							prop = value
						}
					}
				} else if key == "content" {
					content = value
				}
			}
			if content != "" && prop != "" {
				return content
			}
		}
	}
	return ""
}

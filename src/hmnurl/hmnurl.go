package hmnurl

import (
	"fmt"
	"net/url"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

const StaticPath = "/public"
const StaticThemePath = "/public/themes"

type Q struct {
	Name  string
	Value string
}

func QFromURL(u *url.URL) []Q {
	var result []Q
	for key, values := range u.Query() {
		for _, v := range values {
			result = append(result, Q{Name: key, Value: v})
		}
	}
	return result
}

var baseUrlParsed url.URL
var cacheBust string
var isTest bool

func init() {
	SetGlobalBaseUrl(config.Config.BaseUrl)
	SetCacheBust(fmt.Sprint(time.Now().Unix()))
}

func SetGlobalBaseUrl(fullBaseUrl string) {
	parsed, err := url.Parse(fullBaseUrl)
	if err != nil {
		panic(oops.New(err, "could not parse base URL"))
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		panic(oops.New(nil, "Website is misconfigured. Config should include a full BaseUrl (e.g. \"http://handmade.local:9001\")"))
	}

	baseUrlParsed = *parsed
}

func SetCacheBust(newCacheBust string) {
	cacheBust = newCacheBust
}

func Url(path string, query []Q) string {
	return ProjectUrl(path, query, "")
}

func ProjectUrl(path string, query []Q, slug string) string {
	return ProjectUrlWithFragment(path, query, slug, "")
}

func ProjectUrlWithFragment(path string, query []Q, slug string, fragment string) string {
	subdomain := slug
	if slug == models.HMNProjectSlug {
		subdomain = ""
	}

	host := baseUrlParsed.Host
	if len(subdomain) > 0 {
		host = slug + "." + host
	}

	url := url.URL{
		Scheme:   baseUrlParsed.Scheme,
		Host:     host,
		Path:     trim(path),
		RawQuery: encodeQuery(query),
		Fragment: fragment,
	}

	return url.String()
}

func trim(path string) string {
	if len(path) > 0 && path[0] == '/' {
		return path[1:]
	}
	return path
}

func encodeQuery(query []Q) string {
	result := url.Values{}
	for _, q := range query {
		result.Set(q.Name, q.Value)
	}
	return result.Encode()
}

package hmnurl

import (
	"fmt"
	"net/url"
	"regexp"
	"time"

	"git.handmade.network/hmn/hmn/src/models"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/oops"
)

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
var S3BaseUrl string
var isTest bool

func init() {
	SetGlobalBaseUrl(config.Config.BaseUrl)
	SetCacheBust(fmt.Sprint(time.Now().Unix()))
	SetS3BaseUrl(config.Config.DigitalOcean.AssetsPublicUrlRoot)
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

func SetS3BaseUrl(base string) {
	S3BaseUrl = base
	RegexS3Asset = regexp.MustCompile(fmt.Sprintf("%s(?P<key>[\\w\\-./]+)", regexp.QuoteMeta(S3BaseUrl)))
}

func GetBaseHost() string {
	return baseUrlParsed.Host
}

type UrlContext struct {
	PersonalProject bool
	ProjectID       int
	ProjectSlug     string
	ProjectName     string
}

var HMNProjectContext = UrlContext{
	PersonalProject: false,
	ProjectID:       models.HMNProjectID,
	ProjectSlug:     models.HMNProjectSlug,
}

func Url(path string, query []Q) string {
	return UrlWithFragment(path, query, "")
}

func UrlWithFragment(path string, query []Q, fragment string) string {
	return HMNProjectContext.UrlWithFragment(path, query, fragment)
}

// Takes a project URL and rewrites it using the current URL context. This can be used
// to convert a personal project URL to official and vice versa.
func (c *UrlContext) RewriteProjectUrl(u *url.URL) string {
	// we need to strip anything matching the personal project regex to get the base path
	match := RegexPersonalProject.FindString(u.Path)
	return c.Url(u.Path[len(match):], QFromURL(u))
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

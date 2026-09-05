package hmnurl

import (
	"fmt"
	"net/url"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/models"
	// NOTE(ben): Avoid importing oops to keep wasm builds smaller (thanks zerolog)
)

var baseUrlParsed url.URL
var cacheBustVersion string
var S3BaseUrl string

// ----------------------------------------------------------------------------
// Init functions (called primarily from src/config/init.go)

func SetGlobalBaseUrl(fullBaseUrl string) {
	parsed, err := url.Parse(fullBaseUrl)
	if err != nil {
		panic(fmt.Errorf("could not parse base URL: %w", err))
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		panic("Website is misconfigured. Config should include a full BaseUrl (e.g. \"http://handmade.local:9001\")")
	}

	baseUrlParsed = *parsed
}

func SetCacheBustVersion(newCacheBustVersion string) {
	cacheBustVersion = newCacheBustVersion
}

func SetS3BaseUrl(base string) {
	S3BaseUrl = base
	RegexS3Asset = regexp.MustCompile(fmt.Sprintf("%s(?P<key>[\\w\\-./]+)", regexp.QuoteMeta(S3BaseUrl)))
}

// ----------------------------------------------------------------------------
// Actual package

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

func GetBaseHost() string {
	return baseUrlParsed.Host
}

func GetOfficialProjectSlugFromHost(host string) string {
	hostPrefix := strings.TrimSuffix(host, GetBaseHost())
	slug := strings.TrimRight(hostPrefix, ".")
	return slug
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

func (c *UrlContext) IsHMN() bool {
	return c.ProjectID == models.HMNProjectID
}

func Url(path string, query []Q) string {
	return UrlWithFragment(path, query, "")
}

func UrlWithFragment(path string, query []Q, fragment string) string {
	return HMNProjectContext.UrlWithFragment(path, query, fragment)
}

// Cleans up a URL path so it can be compared against a URL regex. Basically this just means
// futzing with slashes.
func NormalizePath(path string) string {
	res := strings.TrimSuffix(path, "/")
	if res == "" {
		res = "/"
	}
	return res
}

func URLMatchesRoute(url *url.URL, route regexp.Regexp) bool {
	path := NormalizePath(url.Path)
	return route.MatchString(path)
}

// Takes a project URL and rewrites it using the current URL context. This can be used
// to convert a personal project URL to official and vice versa.
func (c *UrlContext) RewriteProjectUrl(u *url.URL) string {
	// we need to strip anything matching the personal project regex to get the base path
	match := RegexPersonalProject.FindString(u.Path)
	return c.Url(u.Path[len(match):], QFromURL(u))
}

// Checks if the given URL string belongs to the HMN website.
func UrlIsLocal(urlString string) bool {
	urlParsed, err := url.Parse(urlString)
	if err != nil {
		return false
	}
	return urlParsed.Host == baseUrlParsed.Host || strings.HasSuffix(urlParsed.Host, "."+baseUrlParsed.Host)
}

// Sanitizes a user-controlled redirect URL to avoid spoofing or phishing. If the URL is empty or
// does not belong to the HMN website, it will return BuildHomepage().
func SafeRedirectUrl(redirect string) string {
	if redirect != "" && UrlIsLocal(redirect) {
		return redirect
	} else {
		return BuildHomepage()
	}
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

package hmnurl

import (
	"net/url"
	"regexp"
	"testing"

	"git.handmade.network/hmn/hmn/src/config"
	"github.com/stretchr/testify/assert"
)

func TestUrl(t *testing.T) {
	defer func() {
		SetGlobalBaseUrl(config.Config.BaseUrl)
	}()
	SetGlobalBaseUrl("http://handmade.test")

	t.Run("no query", func(t *testing.T) {
		result := Url("/test/foo", nil)
		assert.Equal(t, "http://handmade.test/test/foo", result)
	})
	t.Run("yes query", func(t *testing.T) {
		result := Url("/test/foo", []Q{{"bar", "baz"}, {"zig??", "zig & zag!!"}})
		assert.Equal(t, "http://handmade.test/test/foo?bar=baz&zig%3F%3F=zig+%26+zag%21%21", result)
	})
}

func TestHomepage(t *testing.T) {
	AssertRegexMatch(t, BuildHomepage(), RegexHomepage, nil)
}

func TestLogin(t *testing.T) {
	AssertRegexMatch(t, BuildLogin(), RegexLogin, nil)
}

func TestLogout(t *testing.T) {
	AssertRegexMatch(t, BuildLogout(), RegexLogout, nil)
}

func TestStaticPages(t *testing.T) {
	AssertRegexMatch(t, BuildManifesto(), RegexManifesto, nil)
	AssertRegexMatch(t, BuildAbout(), RegexAbout, nil)
	AssertRegexMatch(t, BuildCodeOfConduct(), RegexCodeOfConduct, nil)
	AssertRegexMatch(t, BuildCommunicationGuidelines(), RegexCommunicationGuidelines, nil)
	AssertRegexMatch(t, BuildContactPage(), RegexContactPage, nil)
	AssertRegexMatch(t, BuildMonthlyUpdatePolicy(), RegexMonthlyUpdatePolicy, nil)
	AssertRegexMatch(t, BuildProjectSubmissionGuidelines(), RegexProjectSubmissionGuidelines, nil)
}

func TestFeed(t *testing.T) {
	AssertRegexMatch(t, BuildFeed(), RegexFeed, nil)
	assert.Equal(t, BuildFeed(), BuildFeedWithPage(1))
	AssertRegexMatch(t, BuildFeedWithPage(1), RegexFeed, nil)
	AssertRegexMatch(t, "/feed/1", RegexFeed, nil) // NOTE(asaf): We should never build this URL, but we should still accept it.
	AssertRegexMatch(t, BuildFeedWithPage(5), RegexFeed, map[string]string{"page": "5"})
	assert.Panics(t, func() { BuildFeedWithPage(-1) })
	assert.Panics(t, func() { BuildFeedWithPage(0) })
}

func TestForumThread(t *testing.T) {
	AssertRegexMatch(t, BuildForumThread("", nil, 1, 1), RegexForumThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, BuildForumThread("", []string{"wip"}, 1, 2), RegexForumThread, map[string]string{"cats": "forums/wip", "page": "2", "threadid": "1"})
	AssertRegexMatch(t, BuildForumThread("", []string{"sub", "wip"}, 1, 2), RegexForumThread, map[string]string{"cats": "forums/sub/wip", "page": "2", "threadid": "1"})
	AssertSubdomain(t, BuildForumThread("hmn", nil, 1, 1), "")
	AssertSubdomain(t, BuildForumThread("", nil, 1, 1), "")
	AssertSubdomain(t, BuildForumThread("hero", nil, 1, 1), "hero")
	assert.Panics(t, func() { BuildForumThread("", []string{"", "wip"}, 1, 1) })
	assert.Panics(t, func() { BuildForumThread("", []string{" ", "wip"}, 1, 1) })
	assert.Panics(t, func() { BuildForumThread("", []string{"wip/jobs"}, 1, 1) })
}

func TestForumCategory(t *testing.T) {
	AssertRegexMatch(t, BuildForumCategory("", nil, 1), RegexForumCategory, nil)
	AssertRegexMatch(t, BuildForumCategory("", []string{"wip"}, 2), RegexForumCategory, map[string]string{"cats": "forums/wip", "page": "2"})
	AssertRegexMatch(t, BuildForumCategory("", []string{"sub", "wip"}, 2), RegexForumCategory, map[string]string{"cats": "forums/sub/wip", "page": "2"})
	AssertSubdomain(t, BuildForumCategory("hmn", nil, 1), "")
	AssertSubdomain(t, BuildForumCategory("", nil, 1), "")
	AssertSubdomain(t, BuildForumCategory("hero", nil, 1), "hero")
	assert.Panics(t, func() { BuildForumCategory("", []string{"", "wip"}, 1) })
	assert.Panics(t, func() { BuildForumCategory("", []string{" ", "wip"}, 1) })
	assert.Panics(t, func() { BuildForumCategory("", []string{"wip/jobs"}, 1) })
}

func TestProjectCSS(t *testing.T) {
	AssertRegexMatch(t, BuildProjectCSS("000000"), RegexProjectCSS, nil)
}

func TestPublic(t *testing.T) {
	AssertRegexMatch(t, BuildPublic("test"), RegexPublic, nil)
	AssertRegexMatch(t, BuildPublic("/test"), RegexPublic, nil)
	AssertRegexMatch(t, BuildPublic("/test/"), RegexPublic, nil)
	AssertRegexMatch(t, BuildPublic("/test/thing/image.png"), RegexPublic, nil)
	assert.Panics(t, func() { BuildPublic("") })
	assert.Panics(t, func() { BuildPublic("/") })
	assert.Panics(t, func() { BuildPublic("/thing//image.png") })
	assert.Panics(t, func() { BuildPublic("/thing/ /image.png") })
}

func AssertSubdomain(t *testing.T, fullUrl string, expectedSubdomain string) {
	parsed, err := url.Parse(fullUrl)
	ok := assert.Nilf(t, err, "Full url could not be parsed: %s", fullUrl)
	if !ok {
		return
	}

	fullHost := parsed.Host
	if len(expectedSubdomain) == 0 {
		assert.Equal(t, baseUrlParsed.Host, fullHost, "Did not expect a subdomain")
	} else {
		assert.Equalf(t, expectedSubdomain+"."+baseUrlParsed.Host, fullHost, "Subdomain mismatch")
	}
}

func AssertRegexMatch(t *testing.T, fullUrl string, regex *regexp.Regexp, paramsToVerify map[string]string) {
	parsed, err := url.Parse(fullUrl)
	ok := assert.Nilf(t, err, "Full url could not be parsed: %s", fullUrl)
	if !ok {
		return
	}

	requestPath := parsed.Path
	if len(requestPath) == 0 {
		requestPath = "/"
	}
	match := regex.FindStringSubmatch(requestPath)
	assert.NotNilf(t, match, "Url did not match regex: [%s] vs [%s]", requestPath, regex.String())

	if paramsToVerify != nil {
		subexpNames := regex.SubexpNames()
		for i, matchedValue := range match {
			paramName := subexpNames[i]
			expectedValue, ok := paramsToVerify[paramName]
			if ok {
				assert.Equalf(t, expectedValue, matchedValue, "Param mismatch for [%s]", paramName)
				delete(paramsToVerify, paramName)
			}
		}
		if len(paramsToVerify) > 0 {
			unmatchedParams := make([]string, 0, len(paramsToVerify))
			for paramName := range paramsToVerify {
				unmatchedParams = append(unmatchedParams, paramName)
			}
			assert.Fail(t, "Expected match groups not found", unmatchedParams)
		}
	}
}

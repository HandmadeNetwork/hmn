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
	isTest = true

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
	AssertRegexMatch(t, BuildProjectHomepage("hero"), RegexHomepage, nil)
	AssertSubdomain(t, BuildProjectHomepage("hero"), "hero")
}

func TestShowcase(t *testing.T) {
	AssertRegexMatch(t, BuildShowcase(), RegexShowcase, nil)
}

func TestStreams(t *testing.T) {
	AssertRegexMatch(t, BuildStreams(), RegexStreams, nil)
}

func TestSiteMap(t *testing.T) {
	AssertRegexMatch(t, BuildSiteMap(), RegexSiteMap, nil)
}

func TestWhenIsIt(t *testing.T) {
	AssertRegexMatch(t, BuildWhenIsIt(), RegexWhenIsIt, nil)
}

func TestAtomFeed(t *testing.T) {
	AssertRegexMatch(t, BuildAtomFeed(), RegexAtomFeed, nil)
	AssertRegexMatch(t, BuildAtomFeedForProjects(), RegexAtomFeed, map[string]string{"feedtype": "projects"})
	AssertRegexMatch(t, BuildAtomFeedForShowcase(), RegexAtomFeed, map[string]string{"feedtype": "showcase"})

	// NOTE(asaf): The following tests are for backwards compatibity
	AssertRegexMatch(t, "/atom/projects/new", RegexAtomFeed, map[string]string{"feedtype": "projects"})
	AssertRegexMatch(t, "/atom/showcase/new", RegexAtomFeed, map[string]string{"feedtype": "showcase"})
}

func TestLoginAction(t *testing.T) {
	AssertRegexMatch(t, BuildLoginAction(""), RegexLoginAction, nil)
}

func TestLoginPage(t *testing.T) {
	AssertRegexMatch(t, BuildLoginPage(""), RegexLoginPage, nil)
}

func TestLogoutAction(t *testing.T) {
	AssertRegexMatch(t, BuildLogoutAction(""), RegexLogoutAction, nil)
}

func TestRegister(t *testing.T) {
	AssertRegexMatch(t, BuildRegister(), RegexRegister, nil)
}

func TestRegistrationSuccess(t *testing.T) {
	AssertRegexMatch(t, BuildRegistrationSuccess(), RegexRegistrationSuccess, nil)
}

func TestEmailConfirmation(t *testing.T) {
	AssertRegexMatch(t, BuildEmailConfirmation("mruser", "test_token"), RegexEmailConfirmation, map[string]string{"username": "mruser", "token": "test_token"})
}

func TestPasswordReset(t *testing.T) {
	AssertRegexMatch(t, BuildRequestPasswordReset(), RegexRequestPasswordReset, nil)
	AssertRegexMatch(t, BuildPasswordResetSent(), RegexPasswordResetSent, nil)
	AssertRegexMatch(t, BuildDoPasswordReset("user", "token"), RegexDoPasswordReset, map[string]string{"username": "user", "token": "token"})
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

func TestUserProfile(t *testing.T) {
	AssertRegexMatch(t, BuildUserProfile("test"), RegexUserProfile, map[string]string{"username": "test"})
}

func TestUserSettings(t *testing.T) {
	AssertRegexMatch(t, BuildUserSettings("test"), RegexUserSettings, nil)
}

func TestSnippet(t *testing.T) {
	AssertRegexMatch(t, BuildSnippet(15), RegexSnippet, map[string]string{"snippetid": "15"})
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

func TestProjectIndex(t *testing.T) {
	AssertRegexMatch(t, BuildProjectIndex(1), RegexProjectIndex, nil)
	AssertRegexMatch(t, BuildProjectIndex(2), RegexProjectIndex, map[string]string{"page": "2"})
	assert.Panics(t, func() { BuildProjectIndex(0) })
}

func TestProjectNew(t *testing.T) {
	AssertRegexMatch(t, BuildProjectNew(), RegexProjectNew, nil)
}

func TestProjectNotApproved(t *testing.T) {
	AssertRegexMatch(t, BuildProjectNotApproved("test"), RegexProjectNotApproved, map[string]string{"slug": "test"})
}

func TestProjectEdit(t *testing.T) {
	AssertRegexMatch(t, BuildProjectEdit("test", "foo"), RegexProjectEdit, map[string]string{"slug": "test"})
}

func TestPodcast(t *testing.T) {
	AssertRegexMatch(t, BuildPodcast(""), RegexPodcast, nil)
	AssertSubdomain(t, BuildPodcast(""), "")
	AssertSubdomain(t, BuildPodcast("hmn"), "")
	AssertSubdomain(t, BuildPodcast("hero"), "hero")
}

func TestPodcastEdit(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastEdit(""), RegexPodcastEdit, nil)
}

func TestPodcastEpisode(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastEpisode("", "test"), RegexPodcastEpisode, map[string]string{"episodeid": "test"})
}

func TestPodcastEpisodeNew(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastEpisodeNew(""), RegexPodcastEpisodeNew, nil)
}

func TestPodcastEpisodeEdit(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastEpisodeEdit("", "test"), RegexPodcastEpisodeEdit, map[string]string{"episodeid": "test"})
}

func TestPodcastRSS(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastRSS(""), RegexPodcastRSS, nil)
}

func TestForum(t *testing.T) {
	AssertRegexMatch(t, BuildForum("", nil, 1), RegexForum, nil)
	AssertRegexMatch(t, BuildForum("", []string{"wip"}, 2), RegexForum, map[string]string{"subforums": "wip", "page": "2"})
	AssertRegexMatch(t, BuildForum("", []string{"sub", "wip"}, 2), RegexForum, map[string]string{"subforums": "sub/wip", "page": "2"})
	AssertSubdomain(t, BuildForum("hmn", nil, 1), "")
	AssertSubdomain(t, BuildForum("", nil, 1), "")
	AssertSubdomain(t, BuildForum("hero", nil, 1), "hero")
	assert.Panics(t, func() { BuildForum("", nil, 0) })
	assert.Panics(t, func() { BuildForum("", []string{"", "wip"}, 1) })
	assert.Panics(t, func() { BuildForum("", []string{" ", "wip"}, 1) })
	assert.Panics(t, func() { BuildForum("", []string{"wip/jobs"}, 1) })
}

func TestForumNewThread(t *testing.T) {
	AssertRegexMatch(t, BuildForumNewThread("", []string{"sub", "wip"}, false), RegexForumNewThread, map[string]string{"subforums": "sub/wip"})
	AssertRegexMatch(t, BuildForumNewThread("", []string{"sub", "wip"}, true), RegexForumNewThreadSubmit, map[string]string{"subforums": "sub/wip"})
}

func TestForumThread(t *testing.T) {
	AssertRegexMatch(t, BuildForumThread("", nil, 1, "", 1), RegexForumThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, BuildForumThread("", nil, 1, "thread/title/123http://", 2), RegexForumThread, map[string]string{"threadid": "1", "page": "2"})
	AssertRegexMatch(t, BuildForumThreadWithPostHash("", nil, 1, "thread/title/123http://", 2, 123), RegexForumThread, map[string]string{"threadid": "1", "page": "2"})
	AssertSubdomain(t, BuildForumThread("hero", nil, 1, "", 1), "hero")
	assert.Panics(t, func() { BuildForumThread("", nil, -1, "", 1) })
	assert.Panics(t, func() { BuildForumThread("", nil, 1, "", -1) })
}

func TestForumPost(t *testing.T) {
	AssertRegexMatch(t, BuildForumPost("", nil, 1, 2), RegexForumPost, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildForumPost("", nil, 1, 2), RegexForumThread)
	AssertSubdomain(t, BuildForumPost("hero", nil, 1, 2), "hero")
	assert.Panics(t, func() { BuildForumPost("", nil, 1, -1) })
}

func TestForumPostDelete(t *testing.T) {
	AssertRegexMatch(t, BuildForumPostDelete("", nil, 1, 2), RegexForumPostDelete, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildForumPostDelete("", nil, 1, 2), RegexForumPost)
	AssertSubdomain(t, BuildForumPostDelete("hero", nil, 1, 2), "hero")
}

func TestForumPostEdit(t *testing.T) {
	AssertRegexMatch(t, BuildForumPostEdit("", nil, 1, 2), RegexForumPostEdit, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildForumPostEdit("", nil, 1, 2), RegexForumPost)
	AssertSubdomain(t, BuildForumPostEdit("hero", nil, 1, 2), "hero")
}

func TestForumPostReply(t *testing.T) {
	AssertRegexMatch(t, BuildForumPostReply("", nil, 1, 2), RegexForumPostReply, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildForumPostReply("", nil, 1, 2), RegexForumPost)
	AssertSubdomain(t, BuildForumPostReply("hero", nil, 1, 2), "hero")
}

func TestBlog(t *testing.T) {
	AssertRegexMatch(t, BuildBlog("", 1), RegexBlog, nil)
	AssertRegexMatch(t, BuildBlog("", 2), RegexBlog, map[string]string{"page": "2"})
	AssertSubdomain(t, BuildBlog("hero", 1), "hero")
}

func TestBlogThread(t *testing.T) {
	AssertRegexMatch(t, BuildBlogThread("", 1, ""), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, BuildBlogThread("", 1, ""), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, BuildBlogThread("", 1, "title/bla/http://"), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, BuildBlogThreadWithPostHash("", 1, "title/bla/http://", 123), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexNoMatch(t, BuildBlogThread("", 1, ""), RegexBlog)
	AssertSubdomain(t, BuildBlogThread("hero", 1, ""), "hero")
}

func TestBlogPost(t *testing.T) {
	AssertRegexMatch(t, BuildBlogPost("", 1, 2), RegexBlogPost, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildBlogPost("", 1, 2), RegexBlogThread)
	AssertSubdomain(t, BuildBlogPost("hero", 1, 2), "hero")
}

func TestBlogPostDelete(t *testing.T) {
	AssertRegexMatch(t, BuildBlogPostDelete("", 1, 2), RegexBlogPostDelete, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildBlogPostDelete("", 1, 2), RegexBlogPost)
	AssertSubdomain(t, BuildBlogPostDelete("hero", 1, 2), "hero")
}

func TestBlogPostEdit(t *testing.T) {
	AssertRegexMatch(t, BuildBlogPostEdit("", 1, 2), RegexBlogPostEdit, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildBlogPostEdit("", 1, 2), RegexBlogPost)
	AssertSubdomain(t, BuildBlogPostEdit("hero", 1, 2), "hero")
}

func TestBlogPostReply(t *testing.T) {
	AssertRegexMatch(t, BuildBlogPostReply("", 1, 2), RegexBlogPostReply, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildBlogPostReply("", 1, 2), RegexBlogPost)
	AssertSubdomain(t, BuildBlogPostReply("hero", 1, 2), "hero")
}

func TestLibrary(t *testing.T) {
	AssertRegexMatch(t, BuildLibrary(""), RegexLibrary, nil)
	AssertSubdomain(t, BuildLibrary("hero"), "hero")
}

func TestLibraryAll(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryAll(""), RegexLibraryAll, nil)
	AssertSubdomain(t, BuildLibraryAll("hero"), "hero")
}

func TestLibraryTopic(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryTopic("", 1), RegexLibraryTopic, map[string]string{"topicid": "1"})
	AssertSubdomain(t, BuildLibraryTopic("hero", 1), "hero")
}

func TestLibraryResource(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryResource("", 1), RegexLibraryResource, map[string]string{"resourceid": "1"})
	AssertSubdomain(t, BuildLibraryResource("hero", 1), "hero")
}

func TestEpisodeGuide(t *testing.T) {
	AssertRegexMatch(t, BuildEpisodeList("hero", ""), RegexEpisodeList, map[string]string{"topic": ""})
	AssertRegexMatch(t, BuildEpisodeList("hero", "code"), RegexEpisodeList, map[string]string{"topic": "code"})
	AssertSubdomain(t, BuildEpisodeList("hero", "code"), "hero")

	AssertRegexMatch(t, BuildEpisode("hero", "code", "day001"), RegexEpisode, map[string]string{"topic": "code", "episode": "day001"})
	AssertSubdomain(t, BuildEpisode("hero", "code", "day001"), "hero")

	AssertRegexMatch(t, BuildCineraIndex("hero", "code"), RegexCineraIndex, map[string]string{"topic": "code"})
	AssertSubdomain(t, BuildCineraIndex("hero", "code"), "hero")
}

func TestProjectCSS(t *testing.T) {
	AssertRegexMatch(t, BuildProjectCSS("000000"), RegexProjectCSS, nil)
}

func TestPublic(t *testing.T) {
	AssertRegexMatch(t, BuildPublic("test", false), RegexPublic, nil)
	AssertRegexMatch(t, BuildPublic("/test", true), RegexPublic, nil)
	AssertRegexMatch(t, BuildPublic("/test/", false), RegexPublic, nil)
	AssertRegexMatch(t, BuildPublic("/test/thing/image.png", true), RegexPublic, nil)
	assert.Panics(t, func() { BuildPublic("", false) })
	assert.Panics(t, func() { BuildPublic("/", false) })
	assert.Panics(t, func() { BuildPublic("/thing//image.png", false) })
	assert.Panics(t, func() { BuildPublic("/thing/ /image.png", false) })
	assert.Panics(t, func() { BuildPublic("/thing/image.png?hello", false) })

	AssertRegexMatch(t, BuildTheme("test.css", "light", true), RegexPublic, nil)
	AssertRegexMatch(t, BuildUserFile("mylogo.png"), RegexPublic, nil)
}

func TestForumMarkRead(t *testing.T) {
	AssertRegexMatch(t, BuildForumMarkRead(5), RegexForumMarkRead, map[string]string{"sfid": "5"})
}

func AssertSubdomain(t *testing.T, fullUrl string, expectedSubdomain string) {
	t.Helper()

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
	t.Helper()

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

func AssertRegexNoMatch(t *testing.T, fullUrl string, regex *regexp.Regexp) {
	t.Helper()

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
	assert.Nilf(t, match, "Url matched regex: [%s] vs [%s]", requestPath, regex.String())
}

func TestThingsThatDontNeedCoverage(t *testing.T) {
	// look the other way ಠ_ಠ
	BuildPodcastEpisodeFile("foo", "bar")
	BuildS3Asset("ha ha")
}

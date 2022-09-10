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

var hmn = HMNProjectContext
var hero = UrlContext{
	PersonalProject: false,
	ProjectID:       2,
	ProjectSlug:     "hero",
	ProjectName:     "Handmade Hero",
}

func TestHomepage(t *testing.T) {
	AssertRegexMatch(t, BuildHomepage(), RegexHomepage, nil)
	AssertRegexMatch(t, hero.BuildHomepage(), RegexHomepage, nil)
	AssertSubdomain(t, hero.BuildHomepage(), "hero")
}

func TestShowcase(t *testing.T) {
	AssertRegexMatch(t, BuildShowcase(), RegexShowcase, nil)
}

func TestStreams(t *testing.T) {
	AssertRegexMatch(t, BuildStreams(), RegexStreams, nil)
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
	AssertRegexMatch(t, BuildRegister(""), RegexRegister, nil)
}

func TestRegistrationSuccess(t *testing.T) {
	AssertRegexMatch(t, BuildRegistrationSuccess(), RegexRegistrationSuccess, nil)
}

func TestEmailConfirmation(t *testing.T) {
	AssertRegexMatch(t, BuildEmailConfirmation("mruser", "test_token", ""), RegexEmailConfirmation, map[string]string{"username": "mruser", "token": "test_token"})
}

func TestPasswordReset(t *testing.T) {
	AssertRegexMatch(t, BuildRequestPasswordReset(), RegexRequestPasswordReset, nil)
	AssertRegexMatch(t, BuildPasswordResetSent(), RegexPasswordResetSent, nil)
	AssertRegexMatch(t, BuildDoPasswordReset("user", "token"), RegexDoPasswordReset, map[string]string{"username": "user", "token": "token"})
}

func TestStaticPages(t *testing.T) {
	AssertRegexMatch(t, BuildManifesto(), RegexManifesto, nil)
	AssertRegexMatch(t, BuildAbout(), RegexAbout, nil)
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

func TestAdmin(t *testing.T) {
	AssertRegexMatch(t, BuildAdminAtomFeed(), RegexAdminAtomFeed, nil)
	AssertRegexMatch(t, BuildAdminApprovalQueue(), RegexAdminApprovalQueue, nil)
	AssertRegexMatch(t, BuildAdminSetUserStatus(), RegexAdminSetUserStatus, nil)
	AssertRegexMatch(t, BuildAdminNukeUser(), RegexAdminNukeUser, nil)
}

func TestSnippet(t *testing.T) {
	AssertRegexMatch(t, BuildSnippet(15), RegexSnippet, map[string]string{"snippetid": "15"})
}

func TestSnippetSubmit(t *testing.T) {
	AssertRegexMatch(t, BuildSnippetSubmit(), RegexSnippetSubmit, nil)
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

func TestPersonalProject(t *testing.T) {
	AssertRegexMatch(t, BuildPersonalProject(123, "test"), RegexPersonalProject, nil)
}

func TestProjectEdit(t *testing.T) {
	AssertRegexMatch(t, hero.BuildProjectEdit("foo"), RegexProjectEdit, nil)
}

func TestPodcast(t *testing.T) {
	AssertRegexMatch(t, BuildPodcast(), RegexPodcast, nil)
}

func TestPodcastEdit(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastEdit(), RegexPodcastEdit, nil)
}

func TestPodcastEpisode(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastEpisode("test"), RegexPodcastEpisode, map[string]string{"episodeid": "test"})
}

func TestPodcastEpisodeNew(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastEpisodeNew(), RegexPodcastEpisodeNew, nil)
}

func TestPodcastEpisodeEdit(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastEpisodeEdit("test"), RegexPodcastEpisodeEdit, map[string]string{"episodeid": "test"})
}

func TestPodcastRSS(t *testing.T) {
	AssertRegexMatch(t, BuildPodcastRSS(), RegexPodcastRSS, nil)
}

func TestFishbowlIndex(t *testing.T) {
	AssertRegexMatch(t, BuildFishbowlIndex(), RegexFishbowlIndex, nil)
}

func TestFishbowl(t *testing.T) {
	AssertRegexMatch(t, BuildFishbowl("oop"), RegexFishbowl, map[string]string{"slug": "oop"})
	AssertRegexNoMatch(t, BuildFishbowl("oop")+"/otherfiles/whatever", RegexFishbowl)
}

func TestEducationIndex(t *testing.T) {
	AssertRegexMatch(t, BuildEducationIndex(), RegexEducationIndex, nil)
	AssertRegexNoMatch(t, BuildEducationArticle("foo"), RegexEducationIndex)
}

func TestEducationGlossary(t *testing.T) {
	AssertRegexMatch(t, BuildEducationGlossary(""), RegexEducationGlossary, map[string]string{"slug": ""})
	AssertRegexMatch(t, BuildEducationGlossary("foo"), RegexEducationGlossary, map[string]string{"slug": "foo"})
}

func TestEducationArticle(t *testing.T) {
	AssertRegexMatch(t, BuildEducationArticle("foo"), RegexEducationArticle, map[string]string{"slug": "foo"})
}

func TestEducationArticleNew(t *testing.T) {
	AssertRegexMatch(t, BuildEducationArticleNew(), RegexEducationArticleNew, nil)
}

func TestEducationArticleEdit(t *testing.T) {
	AssertRegexMatch(t, BuildEducationArticleEdit("foo"), RegexEducationArticleEdit, map[string]string{"slug": "foo"})
}

func TestEducationArticleDelete(t *testing.T) {
	AssertRegexMatch(t, BuildEducationArticleDelete("foo"), RegexEducationArticleDelete, map[string]string{"slug": "foo"})
}

func TestForum(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildForum(nil, 1), RegexForum, nil)
	AssertRegexMatch(t, hmn.BuildForum([]string{"wip"}, 2), RegexForum, map[string]string{"subforums": "wip", "page": "2"})
	AssertRegexMatch(t, hmn.BuildForum([]string{"sub", "wip"}, 2), RegexForum, map[string]string{"subforums": "sub/wip", "page": "2"})
	AssertSubdomain(t, hmn.BuildForum(nil, 1), "")
	AssertSubdomain(t, hero.BuildForum(nil, 1), "hero")
	assert.Panics(t, func() { hmn.BuildForum(nil, 0) })
	assert.Panics(t, func() { hmn.BuildForum([]string{"", "wip"}, 1) })
	assert.Panics(t, func() { hmn.BuildForum([]string{" ", "wip"}, 1) })
	assert.Panics(t, func() { hmn.BuildForum([]string{"wip/jobs"}, 1) })
}

func TestForumNewThread(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildForumNewThread([]string{"sub", "wip"}, false), RegexForumNewThread, map[string]string{"subforums": "sub/wip"})
	AssertRegexMatch(t, hmn.BuildForumNewThread([]string{"sub", "wip"}, true), RegexForumNewThreadSubmit, map[string]string{"subforums": "sub/wip"})
}

func TestForumThread(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildForumThread(nil, 1, "", 1), RegexForumThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, hmn.BuildForumThread(nil, 1, "thread/title/123http://", 2), RegexForumThread, map[string]string{"threadid": "1", "page": "2"})
	AssertRegexMatch(t, hmn.BuildForumThreadWithPostHash(nil, 1, "thread/title/123http://", 2, 123), RegexForumThread, map[string]string{"threadid": "1", "page": "2"})
	AssertSubdomain(t, hero.BuildForumThread(nil, 1, "", 1), "hero")
	assert.Panics(t, func() { hmn.BuildForumThread(nil, -1, "", 1) })
	assert.Panics(t, func() { hmn.BuildForumThread(nil, 1, "", -1) })
}

func TestForumPost(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildForumPost(nil, 1, 2), RegexForumPost, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, hmn.BuildForumPost(nil, 1, 2), RegexForumThread)
	AssertSubdomain(t, hero.BuildForumPost(nil, 1, 2), "hero")
	assert.Panics(t, func() { hmn.BuildForumPost(nil, 1, -1) })
}

func TestForumPostDelete(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildForumPostDelete(nil, 1, 2), RegexForumPostDelete, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, hmn.BuildForumPostDelete(nil, 1, 2), RegexForumPost)
	AssertSubdomain(t, hero.BuildForumPostDelete(nil, 1, 2), "hero")
}

func TestForumPostEdit(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildForumPostEdit(nil, 1, 2), RegexForumPostEdit, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, hmn.BuildForumPostEdit(nil, 1, 2), RegexForumPost)
	AssertSubdomain(t, hero.BuildForumPostEdit(nil, 1, 2), "hero")
}

func TestForumPostReply(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildForumPostReply(nil, 1, 2), RegexForumPostReply, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, hmn.BuildForumPostReply(nil, 1, 2), RegexForumPost)
	AssertSubdomain(t, hero.BuildForumPostReply(nil, 1, 2), "hero")
}

func TestBlog(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildBlog(1), RegexBlog, nil)
	AssertRegexMatch(t, hmn.BuildBlog(2), RegexBlog, map[string]string{"page": "2"})
	AssertSubdomain(t, hero.BuildBlog(1), "hero")
}

func TestBlogNewThread(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildBlogNewThread(), RegexBlogNewThread, nil)
	AssertSubdomain(t, hmn.BuildBlogNewThread(), "")
	AssertRegexMatch(t, hero.BuildBlogNewThread(), RegexBlogNewThread, nil)
	AssertSubdomain(t, hero.BuildBlogNewThread(), "hero")
}

func TestBlogThread(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildBlogThread(1, ""), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, hmn.BuildBlogThread(1, ""), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, hmn.BuildBlogThread(1, "title/bla/http://"), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, hmn.BuildBlogThreadWithPostHash(1, "title/bla/http://", 123), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexNoMatch(t, hmn.BuildBlogThread(1, ""), RegexBlog)
	AssertSubdomain(t, hero.BuildBlogThread(1, ""), "hero")
}

func TestBlogPost(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildBlogPost(1, 2), RegexBlogPost, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, hmn.BuildBlogPost(1, 2), RegexBlogThread)
	AssertSubdomain(t, hero.BuildBlogPost(1, 2), "hero")
}

func TestBlogPostDelete(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildBlogPostDelete(1, 2), RegexBlogPostDelete, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, hmn.BuildBlogPostDelete(1, 2), RegexBlogPost)
	AssertSubdomain(t, hero.BuildBlogPostDelete(1, 2), "hero")
}

func TestBlogPostEdit(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildBlogPostEdit(1, 2), RegexBlogPostEdit, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, hmn.BuildBlogPostEdit(1, 2), RegexBlogPost)
	AssertSubdomain(t, hero.BuildBlogPostEdit(1, 2), "hero")
}

func TestBlogPostReply(t *testing.T) {
	AssertRegexMatch(t, hmn.BuildBlogPostReply(1, 2), RegexBlogPostReply, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, hmn.BuildBlogPostReply(1, 2), RegexBlogPost)
	AssertSubdomain(t, hero.BuildBlogPostReply(1, 2), "hero")
}

func TestEpisodeGuide(t *testing.T) {
	AssertRegexMatch(t, hero.BuildEpisodeList(""), RegexEpisodeList, map[string]string{"topic": ""})
	AssertRegexMatch(t, hero.BuildEpisodeList("code"), RegexEpisodeList, map[string]string{"topic": "code"})
	AssertSubdomain(t, hero.BuildEpisodeList("code"), "hero")

	AssertRegexMatch(t, hero.BuildEpisode("code", "day001"), RegexEpisode, map[string]string{"topic": "code", "episode": "day001"})
	AssertSubdomain(t, hero.BuildEpisode("code", "day001"), "hero")

	AssertRegexMatch(t, hero.BuildCineraIndex("code"), RegexCineraIndex, map[string]string{"topic": "code"})
	AssertSubdomain(t, hero.BuildCineraIndex("code"), "hero")
}

func TestAssetUpload(t *testing.T) {
	AssertRegexMatch(t, hero.BuildAssetUpload(), RegexAssetUpload, nil)
	AssertSubdomain(t, hero.BuildAssetUpload(), "hero")
}

func TestProjectCSS(t *testing.T) {
	AssertRegexMatch(t, BuildProjectCSS("000000"), RegexProjectCSS, nil)
}

func TestMarkdownWorkerJS(t *testing.T) {
	AssertRegexMatch(t, BuildMarkdownWorkerJS(), RegexMarkdownWorkerJS, nil)
}

func TestAPICheckUsername(t *testing.T) {
	AssertRegexMatch(t, BuildAPICheckUsername(), RegexAPICheckUsername, nil)
}

func TestTwitchEventSubCallback(t *testing.T) {
	AssertRegexMatch(t, BuildTwitchEventSubCallback(), RegexTwitchEventSubCallback, nil)
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
	AssertRegexMatch(t, hero.BuildForumMarkRead(5), RegexForumMarkRead, map[string]string{"sfid": "5"})
	AssertSubdomain(t, hero.BuildForumMarkRead(5), "hero")
}

func TestS3Asset(t *testing.T) {
	AssertRegexMatchFull(t, BuildS3Asset("hello"), RegexS3Asset, map[string]string{"key": "hello"})
}

func TestJamIndex(t *testing.T) {
	AssertRegexMatch(t, BuildJamIndex(), RegexJamIndex, nil)
	AssertSubdomain(t, BuildJamIndex(), "")
}

func TestJamIndex2021(t *testing.T) {
	AssertRegexMatch(t, BuildJamIndex2021(), RegexJamIndex2021, nil)
	AssertSubdomain(t, BuildJamIndex2021(), "")
}

func TestJamIndex2022(t *testing.T) {
	AssertRegexMatch(t, BuildJamIndex2022(), RegexJamIndex2022, nil)
	AssertSubdomain(t, BuildJamIndex2022(), "")
}

func TestJamFeed2022(t *testing.T) {
	AssertRegexMatch(t, BuildJamFeed2022(), RegexJamFeed2022, nil)
	AssertSubdomain(t, BuildJamFeed2022(), "")
}

func TestProjectNewJam(t *testing.T) {
	AssertRegexMatch(t, BuildProjectNewJam(), RegexProjectNew, nil)
	AssertSubdomain(t, BuildProjectNewJam(), "")
}

func TestDiscordOAuthCallback(t *testing.T) {
	AssertRegexMatch(t, BuildDiscordOAuthCallback(), RegexDiscordOAuthCallback, nil)
}

func TestDiscordUnlink(t *testing.T) {
	AssertRegexMatch(t, BuildDiscordUnlink(), RegexDiscordUnlink, nil)
}

func TestDiscordShowcaseBacklog(t *testing.T) {
	AssertRegexMatch(t, BuildDiscordShowcaseBacklog(), RegexDiscordShowcaseBacklog, nil)
}

func TestConferences(t *testing.T) {
	AssertRegexMatch(t, BuildConferences(), RegexConferences, nil)
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

	AssertRegexMatchFull(t, requestPath, regex, paramsToVerify)
}

func AssertRegexMatchFull(t *testing.T, fullUrl string, regex *regexp.Regexp, paramsToVerify map[string]string) {
	t.Helper()

	match := regex.FindStringSubmatch(fullUrl)
	assert.NotNilf(t, match, "Url did not match regex: [%s] vs [%s]", fullUrl, regex.String())

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
	BuildPodcastEpisodeFile("foo")
}

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
	AssertRegexMatch(t, BuildProjectHomepage("hero"), RegexHomepage, nil)
	AssertSubdomain(t, BuildProjectHomepage("hero"), "hero")
}

func TestProjectIndex(t *testing.T) {
	AssertRegexMatch(t, BuildProjectIndex(), RegexProjectIndex, nil)
}

func TestSiteMap(t *testing.T) {
	AssertRegexMatch(t, BuildSiteMap(), RegexSiteMap, nil)
}

func TestAtomFeed(t *testing.T) {
	AssertRegexMatch(t, BuildAtomFeed(), RegexAtomFeed, nil)
}

func TestLoginAction(t *testing.T) {
	AssertRegexMatch(t, BuildLoginAction(""), RegexLoginAction, nil)
}

func TestLoginPage(t *testing.T) {
	AssertRegexMatch(t, BuildLoginPage(""), RegexLoginPage, nil)
}

func TestLogoutAction(t *testing.T) {
	AssertRegexMatch(t, BuildLogoutAction(), RegexLogoutAction, nil)
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

func TestMember(t *testing.T) {
	AssertRegexMatch(t, BuildMember("test"), RegexMember, map[string]string{"member": "test"})
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

func TestForumCategory(t *testing.T) {
	AssertRegexMatch(t, BuildForumCategory("", nil, 1), RegexForumCategory, nil)
	AssertRegexMatch(t, BuildForumCategory("", []string{"wip"}, 2), RegexForumCategory, map[string]string{"cats": "wip", "page": "2"})
	AssertRegexMatch(t, BuildForumCategory("", []string{"sub", "wip"}, 2), RegexForumCategory, map[string]string{"cats": "sub/wip", "page": "2"})
	AssertSubdomain(t, BuildForumCategory("hmn", nil, 1), "")
	AssertSubdomain(t, BuildForumCategory("", nil, 1), "")
	AssertSubdomain(t, BuildForumCategory("hero", nil, 1), "hero")
	assert.Panics(t, func() { BuildForumCategory("", nil, 0) })
	assert.Panics(t, func() { BuildForumCategory("", []string{"", "wip"}, 1) })
	assert.Panics(t, func() { BuildForumCategory("", []string{" ", "wip"}, 1) })
	assert.Panics(t, func() { BuildForumCategory("", []string{"wip/jobs"}, 1) })
}

func TestForumNewThread(t *testing.T) {
	AssertRegexMatch(t, BuildForumNewThread("", []string{"sub", "wip"}), RegexForumNewThread, map[string]string{"cats": "sub/wip"})
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

func TestForumPostQuote(t *testing.T) {
	AssertRegexMatch(t, BuildForumPostQuote("", nil, 1, 2), RegexForumPostQuote, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildForumPostQuote("", nil, 1, 2), RegexForumPost)
	AssertSubdomain(t, BuildForumPostQuote("hero", nil, 1, 2), "hero")
}

func TestBlog(t *testing.T) {
	AssertRegexMatch(t, BuildBlog("", 1), RegexBlog, nil)
	AssertRegexMatch(t, BuildBlog("", 2), RegexBlog, map[string]string{"page": "2"})
	AssertSubdomain(t, BuildBlog("hero", 1), "hero")
}

func TestBlogThread(t *testing.T) {
	AssertRegexMatch(t, BuildBlogThread("", 1, "", 1), RegexBlogThread, map[string]string{"threadid": "1"})
	AssertRegexMatch(t, BuildBlogThread("", 1, "", 2), RegexBlogThread, map[string]string{"threadid": "1", "page": "2"})
	AssertRegexMatch(t, BuildBlogThread("", 1, "title/bla/http://", 2), RegexBlogThread, map[string]string{"threadid": "1", "page": "2"})
	AssertRegexMatch(t, BuildBlogThreadWithPostHash("", 1, "title/bla/http://", 2, 123), RegexBlogThread, map[string]string{"threadid": "1", "page": "2"})
	AssertRegexNoMatch(t, BuildBlogThread("", 1, "", 2), RegexBlog)
	AssertSubdomain(t, BuildBlogThread("hero", 1, "", 1), "hero")
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

func TestBlogPostQuote(t *testing.T) {
	AssertRegexMatch(t, BuildBlogPostQuote("", 1, 2), RegexBlogPostQuote, map[string]string{"threadid": "1", "postid": "2"})
	AssertRegexNoMatch(t, BuildBlogPostQuote("", 1, 2), RegexBlogPost)
	AssertSubdomain(t, BuildBlogPostQuote("hero", 1, 2), "hero")
}

func TestWiki(t *testing.T) {
	AssertRegexMatch(t, BuildWiki(""), RegexWiki, nil)
	AssertSubdomain(t, BuildWiki("hero"), "hero")
}

func TestWikiIndex(t *testing.T) {
	AssertRegexMatch(t, BuildWikiIndex(""), RegexWikiIndex, nil)
	AssertSubdomain(t, BuildWikiIndex("hero"), "hero")
}

func TestWikiArticle(t *testing.T) {
	AssertRegexMatch(t, BuildWikiArticle("", 1, ""), RegexWikiArticle, map[string]string{"articleid": "1"})
	AssertRegexMatch(t, BuildWikiArticle("", 1, "wiki/title/--"), RegexWikiArticle, map[string]string{"articleid": "1"})
	AssertRegexMatch(t, BuildWikiArticleWithSectionName("", 1, "wiki/title/--", "Hello world"), RegexWikiArticle, map[string]string{"articleid": "1"})
	AssertSubdomain(t, BuildWikiArticle("hero", 1, ""), "hero")
}

func TestWikiArticleEdit(t *testing.T) {
	AssertRegexMatch(t, BuildWikiArticleEdit("", 1), RegexWikiArticleEdit, map[string]string{"articleid": "1"})
	AssertSubdomain(t, BuildWikiArticleEdit("hero", 1), "hero")
}

func TestWikiArticleDelete(t *testing.T) {
	AssertRegexMatch(t, BuildWikiArticleDelete("", 1), RegexWikiArticleDelete, map[string]string{"articleid": "1"})
	AssertSubdomain(t, BuildWikiArticleDelete("hero", 1), "hero")
}

func TestWikiArticleHistory(t *testing.T) {
	AssertRegexMatch(t, BuildWikiArticleHistory("", 1, ""), RegexWikiArticleHistory, map[string]string{"articleid": "1"})
	AssertRegexMatch(t, BuildWikiArticleHistory("", 1, "wiki/title/--"), RegexWikiArticleHistory, map[string]string{"articleid": "1"})
	AssertSubdomain(t, BuildWikiArticleHistory("hero", 1, ""), "hero")
}

func TestWikiTalk(t *testing.T) {
	AssertRegexMatch(t, BuildWikiTalk("", 1, ""), RegexWikiTalk, map[string]string{"articleid": "1"})
	AssertRegexMatch(t, BuildWikiTalk("", 1, "wiki/title/--"), RegexWikiTalk, map[string]string{"articleid": "1"})
	AssertSubdomain(t, BuildWikiTalk("hero", 1, ""), "hero")
}

func TestWikiRevision(t *testing.T) {
	AssertRegexMatch(t, BuildWikiRevision("", 1, "", 2), RegexWikiRevision, map[string]string{"articleid": "1", "revisionid": "2"})
	AssertRegexMatch(t, BuildWikiRevision("", 1, "wiki/title/--", 2), RegexWikiRevision, map[string]string{"articleid": "1", "revisionid": "2"})
	AssertSubdomain(t, BuildWikiRevision("hero", 1, "", 2), "hero")
}

func TestWikiDiff(t *testing.T) {
	AssertRegexMatch(t, BuildWikiDiff("", 1, "", 2, 3), RegexWikiDiff, map[string]string{"articleid": "1", "revisionidold": "2", "revisionidnew": "3"})
	AssertRegexMatch(t, BuildWikiDiff("", 1, "wiki/title", 2, 3), RegexWikiDiff, map[string]string{"articleid": "1", "revisionidold": "2", "revisionidnew": "3"})
	AssertSubdomain(t, BuildWikiDiff("hero", 1, "wiki/title", 2, 3), "hero")
}

func TestWikiTalkPost(t *testing.T) {
	AssertRegexMatch(t, BuildWikiTalkPost("", 1, 2), RegexWikiTalkPost, map[string]string{"articleid": "1", "postid": "2"})
	AssertSubdomain(t, BuildWikiTalkPost("hero", 1, 2), "hero")
}

func TestWikiTalkPostDelete(t *testing.T) {
	AssertRegexMatch(t, BuildWikiTalkPostDelete("", 1, 2), RegexWikiTalkPostDelete, map[string]string{"articleid": "1", "postid": "2"})
	AssertSubdomain(t, BuildWikiTalkPostDelete("hero", 1, 2), "hero")
}

func TestWikiTalkPostEdit(t *testing.T) {
	AssertRegexMatch(t, BuildWikiTalkPostEdit("", 1, 2), RegexWikiTalkPostEdit, map[string]string{"articleid": "1", "postid": "2"})
	AssertSubdomain(t, BuildWikiTalkPostEdit("hero", 1, 2), "hero")
}

func TestWikiTalkPostReply(t *testing.T) {
	AssertRegexMatch(t, BuildWikiTalkPostReply("", 1, 2), RegexWikiTalkPostReply, map[string]string{"articleid": "1", "postid": "2"})
	AssertSubdomain(t, BuildWikiTalkPostReply("hero", 1, 2), "hero")
}

func TestWikiTalkPostQuote(t *testing.T) {
	AssertRegexMatch(t, BuildWikiTalkPostQuote("", 1, 2), RegexWikiTalkPostQuote, map[string]string{"articleid": "1", "postid": "2"})
	AssertSubdomain(t, BuildWikiTalkPostQuote("hero", 1, 2), "hero")
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

func TestLibraryDiscussion(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryDiscussion("", 1, 2, 1), RegexLibraryDiscussion, map[string]string{"resourceid": "1", "threadid": "2"})
	AssertRegexMatch(t, BuildLibraryDiscussion("", 1, 2, 3), RegexLibraryDiscussion, map[string]string{"resourceid": "1", "threadid": "2", "page": "3"})
	AssertRegexMatch(t, BuildLibraryDiscussionWithPostHash("", 1, 2, 3, 123), RegexLibraryDiscussion, map[string]string{"resourceid": "1", "threadid": "2", "page": "3"})
	AssertSubdomain(t, BuildLibraryDiscussion("hero", 1, 2, 3), "hero")
}

func TestLibraryPost(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryPost("", 1, 2, 3), RegexLibraryPost, map[string]string{"resourceid": "1", "threadid": "2", "postid": "3"})
	AssertSubdomain(t, BuildLibraryPost("hero", 1, 2, 3), "hero")
}

func TestLibraryPostDelete(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryPostDelete("", 1, 2, 3), RegexLibraryPostDelete, map[string]string{"resourceid": "1", "threadid": "2", "postid": "3"})
	AssertSubdomain(t, BuildLibraryPostDelete("hero", 1, 2, 3), "hero")
}

func TestLibraryPostEdit(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryPostEdit("", 1, 2, 3), RegexLibraryPostEdit, map[string]string{"resourceid": "1", "threadid": "2", "postid": "3"})
	AssertSubdomain(t, BuildLibraryPostEdit("hero", 1, 2, 3), "hero")
}

func TestLibraryPostReply(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryPostReply("", 1, 2, 3), RegexLibraryPostReply, map[string]string{"resourceid": "1", "threadid": "2", "postid": "3"})
	AssertSubdomain(t, BuildLibraryPostReply("hero", 1, 2, 3), "hero")
}

func TestLibraryPostQuote(t *testing.T) {
	AssertRegexMatch(t, BuildLibraryPostQuote("", 1, 2, 3), RegexLibraryPostQuote, map[string]string{"resourceid": "1", "threadid": "2", "postid": "3"})
	AssertSubdomain(t, BuildLibraryPostQuote("hero", 1, 2, 3), "hero")
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

func TestMarkRead(t *testing.T) {
	AssertRegexMatch(t, BuildMarkRead(5), RegexMarkRead, map[string]string{"catid": "5"})
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

func AssertRegexNoMatch(t *testing.T, fullUrl string, regex *regexp.Regexp) {
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

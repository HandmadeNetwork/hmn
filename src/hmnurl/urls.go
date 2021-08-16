package hmnurl

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
)

/*
Any function in this package whose name starts with Build is required to be covered by a test.
This helps ensure that we don't generate URLs that can't be routed.
*/

var RegexOldHome = regexp.MustCompile("^/home$")
var RegexHomepage = regexp.MustCompile("^/$")

func BuildHomepage() string {
	return Url("/", nil)
}

func BuildHomepageWithRegistrationSuccess() string {
	return Url("/", []Q{{Name: "registered", Value: "true"}})
}

func BuildProjectHomepage(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/", nil, projectSlug)
}

var RegexShowcase = regexp.MustCompile("^/showcase$")

func BuildShowcase() string {
	defer CatchPanic()
	return Url("/showcase", nil)
}

var RegexStreams = regexp.MustCompile("^/streams$")

func BuildStreams() string {
	defer CatchPanic()
	return Url("/streams", nil)
}

var RegexSiteMap = regexp.MustCompile("^/sitemap$")

func BuildSiteMap() string {
	defer CatchPanic()
	return Url("/sitemap", nil)
}

// QUESTION(ben): Can we change these routes?

var RegexLoginAction = regexp.MustCompile("^/login$")

func BuildLoginAction(redirectTo string) string {
	defer CatchPanic()
	return Url("/login", []Q{{Name: "redirect", Value: redirectTo}})
}

var RegexLoginPage = regexp.MustCompile("^/login$")

func BuildLoginPage(redirectTo string) string {
	defer CatchPanic()
	return Url("/login", []Q{{Name: "redirect", Value: redirectTo}})
}

var RegexLogoutAction = regexp.MustCompile("^/logout$")

func BuildLogoutAction(redir string) string {
	defer CatchPanic()
	if redir == "" {
		redir = "/"
	}
	return Url("/logout", []Q{{"redirect", redir}})
}

var RegexRegister = regexp.MustCompile("^/register$")

func BuildRegister() string {
	defer CatchPanic()
	return Url("/register", nil)
}

var RegexRegistrationSuccess = regexp.MustCompile("^/registered_successfully$")

func BuildRegistrationSuccess() string {
	defer CatchPanic()
	return Url("/registered_successfully", nil)
}

// TODO(asaf): Delete the old version a bit after launch
var RegexOldEmailConfirmation = regexp.MustCompile(`^/_register/confirm/(?P<username>[\w\ \.\,\-@\+\_]+)/(?P<hash>[\d\w]+)/(?P<nonce>.+)[\/]?$`)
var RegexEmailConfirmation = regexp.MustCompile("^/email_confirmation/(?P<username>[^/]+)/(?P<token>[^/]+)$")

func BuildEmailConfirmation(username, token string) string {
	defer CatchPanic()
	return Url(fmt.Sprintf("/email_confirmation/%s/%s", url.PathEscape(username), token), nil)
}

var RegexPasswordResetRequest = regexp.MustCompile("^/password_reset$")

func BuildPasswordResetRequest() string {
	defer CatchPanic()
	return Url("/password_reset", nil)
}

/*
* Static Pages
 */

var RegexManifesto = regexp.MustCompile("^/manifesto$")

func BuildManifesto() string {
	defer CatchPanic()
	return Url("/manifesto", nil)
}

var RegexAbout = regexp.MustCompile("^/about$")

func BuildAbout() string {
	defer CatchPanic()
	return Url("/about", nil)
}

var RegexCodeOfConduct = regexp.MustCompile("^/code-of-conduct$")

func BuildCodeOfConduct() string {
	defer CatchPanic()
	return Url("/code-of-conduct", nil)
}

var RegexCommunicationGuidelines = regexp.MustCompile("^/communication-guidelines$")

func BuildCommunicationGuidelines() string {
	defer CatchPanic()
	return Url("/communication-guidelines", nil)
}

var RegexContactPage = regexp.MustCompile("^/contact$")

func BuildContactPage() string {
	defer CatchPanic()
	return Url("/contact", nil)
}

var RegexMonthlyUpdatePolicy = regexp.MustCompile("^/monthly-update-policy$")

func BuildMonthlyUpdatePolicy() string {
	defer CatchPanic()
	return Url("/monthly-update-policy", nil)
}

var RegexProjectSubmissionGuidelines = regexp.MustCompile("^/project-guidelines$")

func BuildProjectSubmissionGuidelines() string {
	defer CatchPanic()
	return Url("/project-guidelines", nil)
}

/*
* User
 */

var RegexUserProfile = regexp.MustCompile(`^/m/(?P<username>[^/]+)$`)

func BuildUserProfile(username string) string {
	defer CatchPanic()
	if len(username) == 0 {
		panic(oops.New(nil, "Username must not be blank"))
	}
	return Url("/m/"+url.PathEscape(username), nil)
}

// TODO
func BuildUserSettings(username string) string {
	return ""
}

/*
* Snippets
 */

var RegexSnippet = regexp.MustCompile(`^/snippet/(?P<snippetid>\d+)$`)

func BuildSnippet(snippetId int) string {
	defer CatchPanic()
	return Url("/snippet/"+strconv.Itoa(snippetId), nil)
}

/*
* Feed
 */

var RegexFeed = regexp.MustCompile(`^/feed(/(?P<page>.+)?)?$`)

func BuildFeed() string {
	defer CatchPanic()
	return Url("/feed", nil)
}

func BuildFeedWithPage(page int) string {
	defer CatchPanic()
	if page < 1 {
		panic(oops.New(nil, "Invalid feed page (%d), must be >= 1", page))
	}
	if page == 1 {
		return BuildFeed()
	}
	return Url("/feed/"+strconv.Itoa(page), nil)
}

var RegexAtomFeed = regexp.MustCompile("^/atom(/(?P<feedtype>[^/]+))?(/new)?$") // NOTE(asaf): `/new` for backwards compatibility with old website

func BuildAtomFeed() string {
	defer CatchPanic()
	return Url("/atom", nil)
}

func BuildAtomFeedForProjects() string {
	defer CatchPanic()
	return Url("/atom/projects", nil)
}

func BuildAtomFeedForShowcase() string {
	defer CatchPanic()
	return Url("/atom/showcase", nil)
}

/*
* Projects
 */

var RegexProjectIndex = regexp.MustCompile("^/projects(/(?P<page>.+)?)?$")

func BuildProjectIndex(page int) string {
	defer CatchPanic()
	if page < 1 {
		panic(oops.New(nil, "page must be >= 1"))
	}
	if page == 1 {
		return Url("/projects", nil)
	} else {
		return Url(fmt.Sprintf("/projects/%d", page), nil)
	}
}

var RegexProjectNew = regexp.MustCompile("^/projects/new$")

func BuildProjectNew() string {
	defer CatchPanic()

	return Url("/projects/new", nil)
}

var RegexProjectNotApproved = regexp.MustCompile("^/p/(?P<slug>.+)$")

func BuildProjectNotApproved(slug string) string {
	defer CatchPanic()

	return Url(fmt.Sprintf("/p/%s", slug), nil)
}

var RegexProjectEdit = regexp.MustCompile("^/p/(?P<slug>.+)/edit$")

func BuildProjectEdit(slug string, section string) string {
	defer CatchPanic()

	return ProjectUrlWithFragment(fmt.Sprintf("/p/%s/edit", slug), nil, "", section)
}

/*
* Podcast
 */

var RegexPodcast = regexp.MustCompile(`^/podcast$`)

func BuildPodcast(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/podcast", nil, projectSlug)
}

var RegexPodcastEdit = regexp.MustCompile(`^/podcast/edit$`)

func BuildPodcastEdit(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/podcast/edit", nil, projectSlug)
}

func BuildPodcastEditSuccess(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/podcast/edit", []Q{Q{"success", "true"}}, projectSlug)
}

var RegexPodcastEpisode = regexp.MustCompile(`^/podcast/ep/(?P<episodeid>[^/]+)$`)

func BuildPodcastEpisode(projectSlug string, episodeGUID string) string {
	defer CatchPanic()
	return ProjectUrl(fmt.Sprintf("/podcast/ep/%s", episodeGUID), nil, projectSlug)
}

var RegexPodcastEpisodeNew = regexp.MustCompile(`^/podcast/ep/new$`)

func BuildPodcastEpisodeNew(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/podcast/ep/new", nil, projectSlug)
}

var RegexPodcastEpisodeEdit = regexp.MustCompile(`^/podcast/ep/(?P<episodeid>[^/]+)/edit$`)

func BuildPodcastEpisodeEdit(projectSlug string, episodeGUID string) string {
	defer CatchPanic()
	return ProjectUrl(fmt.Sprintf("/podcast/ep/%s/edit", episodeGUID), nil, projectSlug)
}

func BuildPodcastEpisodeEditSuccess(projectSlug string, episodeGUID string) string {
	defer CatchPanic()
	return ProjectUrl(fmt.Sprintf("/podcast/ep/%s/edit", episodeGUID), []Q{Q{"success", "true"}}, projectSlug)
}

var RegexPodcastRSS = regexp.MustCompile(`^/podcast/podcast.xml$`)

func BuildPodcastRSS(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/podcast/podcast.xml", nil, projectSlug)
}

func BuildPodcastEpisodeFile(projectSlug string, filename string) string {
	defer CatchPanic()
	return BuildUserFile(fmt.Sprintf("podcast/%s/%s", projectSlug, filename))
}

/*
* Forums
 */

// TODO(asaf): This also matches urls generated by BuildForumThread (/t/ is identified as a subforum, and the threadid as a page)
// This shouldn't be a problem since we will match Thread before Subforum in the router, but should we enforce it here?
var RegexForum = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?(/(?P<page>\d+))?$`)

func BuildForum(projectSlug string, subforums []string, page int) string {
	defer CatchPanic()
	if page < 1 {
		panic(oops.New(nil, "Invalid forum thread page (%d), must be >= 1", page))
	}

	builder := buildSubforumPath(subforums)

	if page > 1 {
		builder.WriteRune('/')
		builder.WriteString(strconv.Itoa(page))
	}

	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexForumNewThread = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/new$`)
var RegexForumNewThreadSubmit = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/new/submit$`)

func BuildForumNewThread(projectSlug string, subforums []string, submit bool) string {
	defer CatchPanic()
	builder := buildSubforumPath(subforums)
	builder.WriteString("/t/new")
	if submit {
		builder.WriteString("/submit")
	}

	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexForumThread = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)(-([^/]+))?(/(?P<page>\d+))?$`)

func BuildForumThread(projectSlug string, subforums []string, threadId int, title string, page int) string {
	defer CatchPanic()
	builder := buildForumThreadPath(subforums, threadId, title, page)

	return ProjectUrl(builder.String(), nil, projectSlug)
}

func BuildForumThreadWithPostHash(projectSlug string, subforums []string, threadId int, title string, page int, postId int) string {
	defer CatchPanic()
	builder := buildForumThreadPath(subforums, threadId, title, page)

	return ProjectUrlWithFragment(builder.String(), nil, projectSlug, strconv.Itoa(postId))
}

var RegexForumPost = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)/p/(?P<postid>\d+)$`)

func BuildForumPost(projectSlug string, subforums []string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildForumPostPath(subforums, threadId, postId)

	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexForumPostDelete = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)/p/(?P<postid>\d+)/delete$`)

func BuildForumPostDelete(projectSlug string, subforums []string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildForumPostPath(subforums, threadId, postId)
	builder.WriteString("/delete")
	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexForumPostEdit = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)/p/(?P<postid>\d+)/edit$`)

func BuildForumPostEdit(projectSlug string, subforums []string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildForumPostPath(subforums, threadId, postId)
	builder.WriteString("/edit")
	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexForumPostReply = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)/p/(?P<postid>\d+)/reply$`)

// TODO: It's kinda weird that we have "replies" to a specific post. That's not how the data works. I guess this just affects what you see as the "post you're replying to" on the post composer page?
func BuildForumPostReply(projectSlug string, subforums []string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildForumPostPath(subforums, threadId, postId)
	builder.WriteString("/reply")
	return ProjectUrl(builder.String(), nil, projectSlug)
}

/*
* Blog
 */

var RegexBlogsRedirect = regexp.MustCompile(`^/blogs`)

var RegexBlog = regexp.MustCompile(`^/blog(/(?P<page>\d+))?$`)

func BuildBlog(projectSlug string, page int) string {
	defer CatchPanic()
	if page < 1 {
		panic(oops.New(nil, "Invalid blog page (%d), must be >= 1", page))
	}
	path := "/blog"

	if page > 1 {
		path += "/" + strconv.Itoa(page)
	}

	return ProjectUrl(path, nil, projectSlug)
}

var RegexBlogThread = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)(-([^/]+))?$`)

func BuildBlogThread(projectSlug string, threadId int, title string) string {
	defer CatchPanic()
	builder := buildBlogThreadPath(threadId, title)
	return ProjectUrl(builder.String(), nil, projectSlug)
}

func BuildBlogThreadWithPostHash(projectSlug string, threadId int, title string, postId int) string {
	defer CatchPanic()
	builder := buildBlogThreadPath(threadId, title)
	return ProjectUrlWithFragment(builder.String(), nil, projectSlug, strconv.Itoa(postId))
}

var RegexBlogNewThread = regexp.MustCompile(`^/blog/new$`)

func BuildBlogNewThread(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/blog/new", nil, projectSlug)
}

var RegexBlogPost = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)/e/(?P<postid>\d+)$`)

func BuildBlogPost(projectSlug string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildBlogPostPath(threadId, postId)
	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexBlogPostDelete = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)/e/(?P<postid>\d+)/delete$`)

func BuildBlogPostDelete(projectSlug string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildBlogPostPath(threadId, postId)
	builder.WriteString("/delete")
	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexBlogPostEdit = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)/e/(?P<postid>\d+)/edit$`)

func BuildBlogPostEdit(projectSlug string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildBlogPostPath(threadId, postId)
	builder.WriteString("/edit")
	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexBlogPostReply = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)/e/(?P<postid>\d+)/reply$`)

func BuildBlogPostReply(projectSlug string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildBlogPostPath(threadId, postId)
	builder.WriteString("/reply")
	return ProjectUrl(builder.String(), nil, projectSlug)
}

/*
* Library
 */

var RegexLibrary = regexp.MustCompile(`^/library$`)

func BuildLibrary(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/library", nil, projectSlug)
}

var RegexLibraryAll = regexp.MustCompile(`^/library/all$`)

func BuildLibraryAll(projectSlug string) string {
	defer CatchPanic()
	return ProjectUrl("/library/all", nil, projectSlug)
}

var RegexLibraryTopic = regexp.MustCompile(`^/library/topic/(?P<topicid>\d+)$`)

func BuildLibraryTopic(projectSlug string, topicId int) string {
	defer CatchPanic()
	if topicId < 1 {
		panic(oops.New(nil, "Invalid library topic ID (%d), must be >= 1", topicId))
	}

	var builder strings.Builder
	builder.WriteString("/library/topic/")
	builder.WriteString(strconv.Itoa(topicId))

	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexLibraryResource = regexp.MustCompile(`^/library/resource/(?P<resourceid>\d+)$`)

func BuildLibraryResource(projectSlug string, resourceId int) string {
	defer CatchPanic()
	builder := buildLibraryResourcePath(resourceId)

	return ProjectUrl(builder.String(), nil, projectSlug)
}

/*
* Discord OAuth
 */

var RegexDiscordTest = regexp.MustCompile("^/discord$")

func BuildDiscordTest() string {
	return Url("/discord", nil)
}

var RegexDiscordOAuthCallback = regexp.MustCompile("^/_discord_callback$")

func BuildDiscordOAuthCallback() string {
	return Url("/_discord_callback", nil)
}

var RegexDiscordUnlink = regexp.MustCompile("^/_discord_unlink$")

func BuildDiscordUnlink() string {
	return Url("/_discord_unlink", nil)
}

/*
* Assets
 */

var RegexProjectCSS = regexp.MustCompile("^/assets/project.css$")

func BuildProjectCSS(color string) string {
	defer CatchPanic()
	return Url("/assets/project.css", []Q{{"color", color}})
}

var RegexEditorPreviewsJS = regexp.MustCompile("^/assets/editorpreviews.js$")

func BuildEditorPreviewsJS() string {
	defer CatchPanic()
	return Url("/assets/editorpreviews.js", nil)
}

// NOTE(asaf): No Regex or tests for remote assets, since we don't parse it ourselves
func BuildS3Asset(s3key string) string {
	defer CatchPanic()
	return fmt.Sprintf("%s%s", S3BaseUrl, s3key)
}

var RegexPublic = regexp.MustCompile("^/public/.+$")

func BuildPublic(filepath string, cachebust bool) string {
	defer CatchPanic()
	filepath = strings.Trim(filepath, "/")
	if len(strings.TrimSpace(filepath)) == 0 {
		panic(oops.New(nil, "Attempted to build a /public url with no path"))
	}
	if strings.Contains(filepath, "?") {
		panic(oops.New(nil, "Public url failpath must not contain query params"))
	}
	var builder strings.Builder
	builder.WriteString("/public")
	pathParts := strings.Split(filepath, "/")
	for _, part := range pathParts {
		part = strings.TrimSpace(part)
		if len(part) == 0 {
			panic(oops.New(nil, "Attempted to build a /public url with blank path segments: %s", filepath))
		}
		builder.WriteRune('/')
		builder.WriteString(part)
	}
	var query []Q
	if cachebust {
		query = []Q{{"v", cacheBust}}
	}
	return Url(builder.String(), query)
}

func BuildTheme(filepath string, theme string, cachebust bool) string {
	defer CatchPanic()
	filepath = strings.Trim(filepath, "/")
	if len(theme) == 0 {
		panic(oops.New(nil, "Theme can't be blank"))
	}
	return BuildPublic(fmt.Sprintf("themes/%s/%s", theme, filepath), cachebust)
}

func BuildUserFile(filepath string) string {
	filepath = strings.Trim(filepath, "/")
	return BuildPublic(fmt.Sprintf("media/%s", filepath), false)
}

/*
* Other
 */

var RegexForumMarkRead = regexp.MustCompile(`^/markread/(?P<sfid>\d+)$`)

// NOTE(asaf): subforumId == 0 means ALL SUBFORUMS
func BuildForumMarkRead(subforumId int) string {
	defer CatchPanic()
	if subforumId < 0 {
		panic(oops.New(nil, "Invalid subforum ID (%d), must be >= 0", subforumId))
	}

	var builder strings.Builder
	builder.WriteString("/markread/")
	builder.WriteString(strconv.Itoa(subforumId))

	return Url(builder.String(), nil)
}

var RegexCatchAll = regexp.MustCompile("")

/*
* Helper functions
 */

func buildSubforumPath(subforums []string) *strings.Builder {
	for _, subforum := range subforums {
		if strings.Contains(subforum, "/") {
			panic(oops.New(nil, "Tried building forum url with / in subforum name"))
		}
		subforum = strings.TrimSpace(subforum)
		if len(subforum) == 0 {
			panic(oops.New(nil, "Tried building forum url with blank subforum"))
		}
	}

	var builder strings.Builder
	builder.WriteString("/forums")
	for _, subforum := range subforums {
		builder.WriteRune('/')
		builder.WriteString(subforum)
	}

	return &builder
}

func buildForumThreadPath(subforums []string, threadId int, title string, page int) *strings.Builder {
	if page < 1 {
		panic(oops.New(nil, "Invalid forum thread page (%d), must be >= 1", page))
	}

	if threadId < 1 {
		panic(oops.New(nil, "Invalid forum thread ID (%d), must be >= 1", threadId))
	}

	builder := buildSubforumPath(subforums)

	builder.WriteString("/t/")
	builder.WriteString(strconv.Itoa(threadId))
	if len(title) > 0 {
		builder.WriteRune('-')
		builder.WriteString(PathSafeTitle(title))
	}
	if page > 1 {
		builder.WriteRune('/')
		builder.WriteString(strconv.Itoa(page))
	}

	return builder
}

func buildForumPostPath(subforums []string, threadId int, postId int) *strings.Builder {
	if threadId < 1 {
		panic(oops.New(nil, "Invalid forum thread ID (%d), must be >= 1", threadId))
	}

	if postId < 1 {
		panic(oops.New(nil, "Invalid forum post ID (%d), must be >= 1", postId))
	}

	builder := buildSubforumPath(subforums)

	builder.WriteString("/t/")
	builder.WriteString(strconv.Itoa(threadId))
	builder.WriteString("/p/")
	builder.WriteString(strconv.Itoa(postId))

	return builder
}

func buildBlogThreadPath(threadId int, title string) *strings.Builder {
	if threadId < 1 {
		panic(oops.New(nil, "Invalid blog thread ID (%d), must be >= 1", threadId))
	}

	var builder strings.Builder

	builder.WriteString("/blog/p/")
	builder.WriteString(strconv.Itoa(threadId))
	if len(title) > 0 {
		builder.WriteRune('-')
		builder.WriteString(PathSafeTitle(title))
	}

	return &builder
}

func buildBlogPostPath(threadId int, postId int) *strings.Builder {
	if threadId < 1 {
		panic(oops.New(nil, "Invalid blog thread ID (%d), must be >= 1", threadId))
	}

	if postId < 1 {
		panic(oops.New(nil, "Invalid blog post ID (%d), must be >= 1", postId))
	}

	var builder strings.Builder

	builder.WriteString("/blog/p/")
	builder.WriteString(strconv.Itoa(threadId))
	builder.WriteString("/e/")
	builder.WriteString(strconv.Itoa(postId))

	return &builder
}

func buildLibraryResourcePath(resourceId int) *strings.Builder {
	if resourceId < 1 {
		panic(oops.New(nil, "Invalid library resource ID (%d), must be >= 1", resourceId))
	}

	var builder strings.Builder
	builder.WriteString("/library/resource/")
	builder.WriteString(strconv.Itoa(resourceId))

	return &builder
}

func buildLibraryDiscussionPath(resourceId int, threadId int, page int) *strings.Builder {
	if page < 1 {
		panic(oops.New(nil, "Invalid page number (%d), must be >= 1", page))
	}
	if threadId < 1 {
		panic(oops.New(nil, "Invalid library thread ID (%d), must be >= 1", threadId))
	}
	builder := buildLibraryResourcePath(resourceId)
	builder.WriteString("/d/")
	builder.WriteString(strconv.Itoa(threadId))
	if page > 1 {
		builder.WriteRune('/')
		builder.WriteString(strconv.Itoa(page))
	}
	return builder
}

func buildLibraryPostPath(resourceId int, threadId int, postId int) *strings.Builder {
	if threadId < 1 {
		panic(oops.New(nil, "Invalid library thread ID (%d), must be >= 1", threadId))
	}
	if postId < 1 {
		panic(oops.New(nil, "Invalid library post ID (%d), must be >= 1", postId))
	}
	builder := buildLibraryResourcePath(resourceId)
	builder.WriteString("/d/")
	builder.WriteString(strconv.Itoa(threadId))
	builder.WriteString("/p/")
	builder.WriteString(strconv.Itoa(postId))
	return builder
}

var PathCharsToClear = regexp.MustCompile("[$&`<>{}()\\[\\]\"+#%@;=?\\\\^|~‘]")
var PathCharsToReplace = regexp.MustCompile("[ :/\\\\]")

func PathSafeTitle(title string) string {
	title = strings.ToLower(title)
	title = PathCharsToReplace.ReplaceAllLiteralString(title, "_")
	title = PathCharsToClear.ReplaceAllLiteralString(title, "")
	title = url.PathEscape(title)
	return title
}

// TODO(asaf): Find a nicer solution that doesn't require adding a defer to every construction function while also not printing errors in tests.
func CatchPanic() {
	if !isTest {
		if recovered := recover(); recovered != nil {
			logging.LogPanicValue(nil, recovered, "Url construction failed")
		}
	}
}

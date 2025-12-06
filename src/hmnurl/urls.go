package hmnurl

import (
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

/*
Any function in this package whose name starts with Build is required to be covered by a test.
This helps ensure that we don't generate URLs that can't be routed.
*/

var RegexOldHome = regexp.MustCompile("^/home$")
var RegexHomepage = regexp.MustCompile("^/$")

func BuildHomepage() string {
	return HMNProjectContext.BuildHomepage()
}

func (c *UrlContext) BuildHomepage() string {
	return c.Url("/", nil)
}

var RegexJamsIndex = regexp.MustCompile("^/jams$")

func BuildJamsIndex() string {
	defer CatchPanic()
	return Url("/jams", nil)
}

var RegexJamIndex = regexp.MustCompile("^/jam$")

func BuildJamIndex() string {
	defer CatchPanic()
	return Url("/jam", nil)
}

var RegexJamIndex2021 = regexp.MustCompile("^/jam/2021$")

func BuildJamIndex2021() string {
	defer CatchPanic()
	return Url("/jam/2021", nil)
}

var RegexJamIndex2022 = regexp.MustCompile("^/jam/2022$")

func BuildJamIndex2022() string {
	defer CatchPanic()
	return Url("/jam/2022", nil)
}

var RegexJamFeed2022 = regexp.MustCompile("^/jam/2022/feed$")

func BuildJamFeed2022() string {
	defer CatchPanic()
	return Url("/jam/2022/feed", nil)
}

var RegexJamIndex2023 = regexp.MustCompile("^/jam/2023$")

func BuildJamIndex2023() string {
	defer CatchPanic()
	return Url("/jam/2023", nil)
}

var RegexJamFeed2023 = regexp.MustCompile("^/jam/2023/feed$")

func BuildJamFeed2023() string {
	defer CatchPanic()
	return Url("/jam/2023/feed", nil)
}

var RegexJamIndex2023_Visibility = regexp.MustCompile("^/jam/visibility-2023$")

func BuildJamIndex2023_Visibility() string {
	defer CatchPanic()
	return Url("/jam/visibility-2023", nil)
}

var RegexJamFeed2023_Visibility = regexp.MustCompile("^/jam/visibility-2023/feed$")

func BuildJamFeed2023_Visibility() string {
	defer CatchPanic()
	return Url("/jam/visibility-2023/feed", nil)
}

var RegexJamRecap2023_Visibility = regexp.MustCompile("^/jam/visibility-2023/recap$")

func BuildJamRecap2023_Visibility() string {
	defer CatchPanic()
	return Url("/jam/visibility-2023/recap", nil)
}

var RegexJamIndex2024_Learning = regexp.MustCompile("^/jam/learning-2024$")

func BuildJamIndex2024_Learning() string {
	defer CatchPanic()
	return Url("/jam/learning-2024", nil)
}

var RegexJamFeed2024_Learning = regexp.MustCompile("^/jam/learning-2024/feed$")

func BuildJamFeed2024_Learning() string {
	defer CatchPanic()
	return Url("/jam/learning-2024/feed", nil)
}

var RegexJamGuidelines2024_Learning = regexp.MustCompile("^/jam/learning-2024/guidelines$")

func BuildJamGuidelines2024_Learning() string {
	defer CatchPanic()
	return Url("/jam/learning-2024/guidelines", nil)
}

var RegexJamIndex2024_Visibility = regexp.MustCompile("^/jam/visibility-2024$")

func BuildJamIndex2024_Visibility() string {
	defer CatchPanic()
	return Url("/jam/visibility-2024", nil)
}

var RegexJamFeed2024_Visibility = regexp.MustCompile("^/jam/visibility-2024/feed$")

func BuildJamFeed2024_Visibility() string {
	defer CatchPanic()
	return Url("/jam/visibility-2024/feed", nil)
}

var RegexJamGuidelines2024_Visibility = regexp.MustCompile("^/jam/visibility-2024/guidelines$")

func BuildJamGuidelines2024_Visibility() string {
	defer CatchPanic()
	return Url("/jam/visibility-2024/guidelines", nil)
}

var RegexJamIndex2024_WRJ = regexp.MustCompile("^/jam/wheel-reinvention-2024$")

func BuildJamIndex2024_WRJ() string {
	defer CatchPanic()
	return Url("/jam/wheel-reinvention-2024", nil)
}

var RegexJamFeed2024_WRJ = regexp.MustCompile("^/jam/wheel-reinvention-2024/feed$")

func BuildJamFeed2024_WRJ() string {
	defer CatchPanic()
	return Url("/jam/wheel-reinvention-2024/feed", nil)
}

var RegexJamGuidelines2024_WRJ = regexp.MustCompile("^/jam/wheel-reinvention-2024/guidelines$")

func BuildJamGuidelines2024_WRJ() string {
	defer CatchPanic()
	return Url("/jam/wheel-reinvention-2024/guidelines", nil)
}

var RegexJamIndex2025_XRay = regexp.MustCompile("^/jam/x-ray-2025$")

func BuildJamIndex2025_XRay() string {
	defer CatchPanic()
	return Url("/jam/x-ray-2025", nil)
}

var RegexJamFeed2025_XRay = regexp.MustCompile("^/jam/x-ray-2025/feed$")

func BuildJamFeed2025_XRay() string {
	defer CatchPanic()
	return Url("/jam/x-ray-2025/feed", nil)
}

var RegexJamGuidelines2025_XRay = regexp.MustCompile("^/jam/x-ray-2025/guidelines$")

func BuildJamGuidelines2025_XRay() string {
	defer CatchPanic()
	return Url("/jam/x-ray-2025/guidelines", nil)
}

func BuildJamIndexAny(slug string) string {
	defer CatchPanic()
	return Url(fmt.Sprintf("/jam/%s", slug), nil)
}

var RegexTimeMachine = regexp.MustCompile("^/timemachine$")

func BuildTimeMachine() string {
	defer CatchPanic()
	return Url("/timemachine", nil)
}

var RegexTimeMachineSubmissions = regexp.MustCompile("^/timemachine/submissions$")

func BuildTimeMachineSubmissions() string {
	defer CatchPanic()
	return Url("/timemachine/submissions", nil)
}

func BuildTimeMachineSubmission(id int) string {
	defer CatchPanic()
	return UrlWithFragment("/timemachine/submissions", nil, strconv.Itoa(id))
}

var RegexTimeMachineAtomFeed = regexp.MustCompile("^/timemachine/submissions/atom$")

func BuildTimeMachineAtomFeed() string {
	defer CatchPanic()
	return Url("/timemachine/submissions/atom", nil)
}

var RegexTimeMachineForm = regexp.MustCompile("^/timemachine/submit$")

func BuildTimeMachineForm() string {
	defer CatchPanic()
	return Url("/timemachine/submit", nil)
}

var RegexTimeMachineFormDone = regexp.MustCompile("^/timemachine/thanks$")

func BuildTimeMachineFormDone() string {
	defer CatchPanic()
	return Url("/timemachine/thanks", nil)
}

var RegexCalendarIndex = regexp.MustCompile("^/calendar$")

func BuildCalendarIndex() string {
	defer CatchPanic()
	return Url("/calendar", nil)
}

var RegexCalendarICal = regexp.MustCompile("^/Handmade Network.ical$")

func BuildCalendarICal() string {
	defer CatchPanic()
	return Url("/Handmade Network.ical", nil)
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
	var q []Q
	if redirectTo != "" {
		q = append(q, Q{Name: "redirect", Value: redirectTo})
	}
	return Url("/login", q)
}

var RegexLoginWithDiscord = regexp.MustCompile("^/login-with-discord$")

func BuildLoginWithDiscord(redirectTo string) string {
	defer CatchPanic()
	return Url("/login-with-discord", []Q{{Name: "redirect", Value: redirectTo}})
}

var RegexLogout = regexp.MustCompile("^/logout$")

func BuildLogoutAction(redir string) string {
	defer CatchPanic()
	if redir == "" {
		redir = "/"
	}
	return Url("/logout", []Q{{"redirect", redir}})
}

var RegexRegister = regexp.MustCompile("^/register$")

func BuildRegister(destination string) string {
	defer CatchPanic()
	var query []Q
	if destination != "" {
		query = append(query, Q{"destination", destination})
	}
	return Url("/register", query)
}

var RegexRegistrationSuccess = regexp.MustCompile("^/registered_successfully$")

func BuildRegistrationSuccess() string {
	defer CatchPanic()
	return Url("/registered_successfully", nil)
}

var RegexEmailConfirmation = regexp.MustCompile("^/email_confirmation/(?P<username>[^/]+)/(?P<token>[^/]+)$")

func BuildEmailConfirmation(username, token string, destination string) string {
	defer CatchPanic()
	var query []Q
	if destination != "" {
		query = append(query, Q{"destination", destination})
	}
	return Url(fmt.Sprintf("/email_confirmation/%s/%s", url.PathEscape(username), token), query)
}

var RegexRequestPasswordReset = regexp.MustCompile("^/password_reset$")

func BuildRequestPasswordReset() string {
	defer CatchPanic()
	return Url("/password_reset", nil)
}

var RegexPasswordResetSent = regexp.MustCompile("^/password_reset/sent$")

func BuildPasswordResetSent() string {
	defer CatchPanic()
	return Url("/password_reset/sent", nil)
}

var RegexOldDoPasswordReset = regexp.MustCompile(`^_password_reset/(?P<username>[\w\ \.\,\-@\+\_]+)/(?P<token>[\d\w]+)[\/]?$`)
var RegexDoPasswordReset = regexp.MustCompile("^/password_reset/(?P<username>[^/]+)/(?P<token>[^/]+)$")

func BuildDoPasswordReset(username string, token string) string {
	defer CatchPanic()
	return Url(fmt.Sprintf("/password_reset/%s/%s", url.PathEscape(username), token), nil)
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

var RegexNewsletterSignup = regexp.MustCompile("^/newsletter$")

func BuildNewsletterSignup() string {
	defer CatchPanic()
	return Url("/newsletter", nil)
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
	return Url("/m/"+username, nil)
}

var RegexUserSettings = regexp.MustCompile(`^/settings$`)

func BuildUserSettings(section string) string {
	return UrlWithFragment("/settings", nil, section)
}

/*
* Admin
 */

var RegexAdminAtomFeed = regexp.MustCompile(`^/admin/atom$`)

func BuildAdminAtomFeed() string {
	defer CatchPanic()
	return Url("/admin/atom", nil)
}

var RegexAdminApprovalQueue = regexp.MustCompile(`^/admin/approvals$`)

func BuildAdminApprovalQueue() string {
	defer CatchPanic()
	return Url("/admin/approvals", nil)
}

var RegexAdminSetUserOptions = regexp.MustCompile(`^/admin/setuseroptions$`)

func BuildAdminSetUserOptions() string {
	defer CatchPanic()
	return Url("/admin/setuseroptions", nil)
}

var RegexAdminNukeUser = regexp.MustCompile(`^/admin/nukeuser$`)

func BuildAdminNukeUser() string {
	defer CatchPanic()
	return Url("/admin/nukeuser", nil)
}

/*
* Snippets
 */

var RegexSnippet = regexp.MustCompile(`^/snippet/(?P<snippetid>\d+)$`)

func BuildSnippet(snippetId int) string {
	defer CatchPanic()
	return Url("/snippet/"+strconv.Itoa(snippetId), nil)
}

var RegexSnippetSubmit = regexp.MustCompile(`^/snippet$`)

func BuildSnippetSubmit() string {
	defer CatchPanic()
	return Url("/snippet", nil)
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

var RegexProjectIndex = regexp.MustCompile(`^/projects$`)

func BuildProjectIndex() string {
	defer CatchPanic()
	return Url("/projects", nil)
}

var RegexProjectNew = regexp.MustCompile("^/p/new$")

func BuildProjectNew() string {
	defer CatchPanic()

	return Url("/p/new", nil)
}

func BuildProjectNewJam() string {
	defer CatchPanic()

	return Url("/p/new", []Q{{Name: "jam", Value: "1"}})
}

var RegexPersonalProject = regexp.MustCompile("^/p/(?P<projectid>[0-9]+)(/(?P<projectslug>[a-zA-Z0-9-]+))?")

func BuildPersonalProject(id int, slug string) string {
	defer CatchPanic()
	return Url(fmt.Sprintf("/p/%d/%s", id, slug), nil)
}

var RegexProjectEdit = regexp.MustCompile("^/edit$")

func (c *UrlContext) BuildProjectEdit(section string) string {
	defer CatchPanic()

	return c.UrlWithFragment("/edit", nil, section)
}

/*
* Podcast
 */

var RegexPodcast = regexp.MustCompile(`^/podcast$`)

func BuildPodcast() string {
	defer CatchPanic()
	return Url("/podcast", nil)
}

var RegexPodcastEdit = regexp.MustCompile(`^/podcast/edit$`)

func BuildPodcastEdit() string {
	defer CatchPanic()
	return Url("/podcast/edit", nil)
}

var RegexPodcastEpisode = regexp.MustCompile(`^/podcast/ep/(?P<episodeid>[^/]+)$`)

func BuildPodcastEpisode(episodeGUID string) string {
	defer CatchPanic()
	return Url(fmt.Sprintf("/podcast/ep/%s", episodeGUID), nil)
}

var RegexPodcastEpisodeNew = regexp.MustCompile(`^/podcast/ep/new$`)

func BuildPodcastEpisodeNew() string {
	defer CatchPanic()
	return Url("/podcast/ep/new", nil)
}

var RegexPodcastEpisodeEdit = regexp.MustCompile(`^/podcast/ep/(?P<episodeid>[^/]+)/edit$`)

func BuildPodcastEpisodeEdit(episodeGUID string) string {
	defer CatchPanic()
	return Url(fmt.Sprintf("/podcast/ep/%s/edit", episodeGUID), nil)
}

var RegexPodcastRSS = regexp.MustCompile(`^/podcast/podcast.xml$`)

func BuildPodcastRSS() string {
	defer CatchPanic()
	return Url("/podcast/podcast.xml", nil)
}

func BuildPodcastEpisodeFile(filename string) string {
	defer CatchPanic()
	return BuildUserFile(fmt.Sprintf("podcast/%s/%s", models.HMNProjectSlug, filename))
}

/*
 * Fishbowls
 */

var RegexFishbowlIndex = regexp.MustCompile(`^/fishbowl$`)

func BuildFishbowlIndex() string {
	defer CatchPanic()
	return Url("/fishbowl", nil)
}

var RegexFishbowl = regexp.MustCompile(`^/fishbowl/(?P<slug>[^/]+)/?$`)

func BuildFishbowl(slug string) string {
	defer CatchPanic()
	return Url(fmt.Sprintf("/fishbowl/%s/", slug), nil)
}

var RegexFishbowlFiles = regexp.MustCompile(`^/fishbowl/(?P<slug>[^/]+)(?P<path>/.+)$`)

/*
 * Education
 */

var RegexEducationIndex = regexp.MustCompile(`^/education$`)

func BuildEducationIndex() string {
	defer CatchPanic()
	return Url("/education", nil)
}

var RegexEducationGlossary = regexp.MustCompile(`^/education/glossary(/(?P<slug>[^/]+))?$`)

func BuildEducationGlossary(termSlug string) string {
	defer CatchPanic()

	if termSlug == "" {
		return Url("/education/glossary", nil)
	} else {
		return Url(fmt.Sprintf("/education/glossary/%s", termSlug), nil)
	}
}

var RegexEducationArticle = regexp.MustCompile(`^/education/(?P<slug>[^/]+)$`)

func BuildEducationArticle(slug string) string {
	return Url(fmt.Sprintf("/education/%s", slug), nil)
}

var RegexEducationArticleNew = regexp.MustCompile(`^/education/new$`)

func BuildEducationArticleNew() string {
	return Url("/education/new", nil)
}

var RegexEducationArticleEdit = regexp.MustCompile(`^/education/(?P<slug>[^/]+)/edit$`)

func BuildEducationArticleEdit(slug string) string {
	return Url(fmt.Sprintf("/education/%s/edit", slug), nil)
}

var RegexEducationArticleDelete = regexp.MustCompile(`^/education/(?P<slug>[^/]+)/delete$`)

func BuildEducationArticleDelete(slug string) string {
	return Url(fmt.Sprintf("/education/%s/delete", slug), nil)
}

var RegexEducationRerender = regexp.MustCompile(`^/education/rerender$`)

func BuildEducationRerender() string {
	return Url("/education/rerender", nil)
}

/*
 * Style test
 */

var RegexStyleTest = regexp.MustCompile(`^/debug/styles$`)

/*
* Forums
 */

// NOTE(asaf): This also matches urls generated by BuildForumThread (/t/ is identified as a subforum, and the threadid as a page)
// Make sure to match Thread before Subforum in the router.
var RegexForum = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?(/(?P<page>\d+))?$`)

func (c *UrlContext) Url(path string, query []Q) string {
	return c.UrlWithFragment(path, query, "")
}

func (c *UrlContext) UrlWithFragment(path string, query []Q, fragment string) string {
	if c == nil {
		logging.Warn().Stack().Msg("URL context was nil; defaulting to the HMN URL context")
		c = &HMNProjectContext
	}

	if c.PersonalProject {
		url := url.URL{
			Scheme:   baseUrlParsed.Scheme,
			Host:     baseUrlParsed.Host,
			Path:     fmt.Sprintf("p/%d/%s/%s", c.ProjectID, models.GeneratePersonalProjectSlug(c.ProjectName), trim(path)),
			RawQuery: encodeQuery(query),
			Fragment: fragment,
		}

		return url.String()
	} else {
		subdomain := c.ProjectSlug
		if c.ProjectSlug == models.HMNProjectSlug {
			subdomain = ""
		}

		host := baseUrlParsed.Host
		if len(subdomain) > 0 {
			host = c.ProjectSlug + "." + host
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
}

func (c *UrlContext) BuildForum(subforums []string, page int) string {
	defer CatchPanic()
	if page < 1 {
		panic(oops.New(nil, "Invalid forum thread page (%d), must be >= 1", page))
	}

	builder := buildSubforumPath(subforums)

	if page > 1 {
		builder.WriteRune('/')
		builder.WriteString(strconv.Itoa(page))
	}

	return c.Url(builder.String(), nil)
}

var RegexForumNewThread = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/new$`)
var RegexForumNewThreadSubmit = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/new/submit$`)

func (c *UrlContext) BuildForumNewThread(subforums []string, submit bool) string {
	defer CatchPanic()
	builder := buildSubforumPath(subforums)
	builder.WriteString("/t/new")
	if submit {
		builder.WriteString("/submit")
	}

	return c.Url(builder.String(), nil)
}

var RegexForumThread = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)(-([^/]+))?(/(?P<page>\d+))?$`)

func (c *UrlContext) BuildForumThread(subforums []string, threadId int, title string, page int) string {
	defer CatchPanic()
	builder := buildForumThreadPath(subforums, threadId, title, page)

	return c.Url(builder.String(), nil)
}

func (c *UrlContext) BuildForumThreadWithPostHash(subforums []string, threadId int, title string, page int, postId int) string {
	defer CatchPanic()
	builder := buildForumThreadPath(subforums, threadId, title, page)

	return c.UrlWithFragment(builder.String(), nil, strconv.Itoa(postId))
}

var RegexForumPost = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)/p/(?P<postid>\d+)$`)

func (c *UrlContext) BuildForumPost(subforums []string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildForumPostPath(subforums, threadId, postId)

	return c.Url(builder.String(), nil)
}

var RegexForumPostDelete = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)/p/(?P<postid>\d+)/delete$`)

func (c *UrlContext) BuildForumPostDelete(subforums []string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildForumPostPath(subforums, threadId, postId)
	builder.WriteString("/delete")
	return c.Url(builder.String(), nil)
}

var RegexForumPostEdit = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)/p/(?P<postid>\d+)/edit$`)

func (c *UrlContext) BuildForumPostEdit(subforums []string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildForumPostPath(subforums, threadId, postId)
	builder.WriteString("/edit")
	return c.Url(builder.String(), nil)
}

var RegexForumPostReply = regexp.MustCompile(`^/forums(/(?P<subforums>[^\d/]+(/[^\d]+)*))?/t/(?P<threadid>\d+)/p/(?P<postid>\d+)/reply$`)

func (c *UrlContext) BuildForumPostReply(subforums []string, threadId int, postId int) string {
	defer CatchPanic()
	builder := buildForumPostPath(subforums, threadId, postId)
	builder.WriteString("/reply")
	return c.Url(builder.String(), nil)
}

var RegexWikiArticle = regexp.MustCompile(`^/wiki/(?P<threadid>\d+)(-([^/]+))?$`)

/*
* Blog
 */

var RegexBlogsRedirect = regexp.MustCompile(`^/blogs(?P<remainder>.*)`)

var RegexBlog = regexp.MustCompile(`^/blog(/(?P<page>\d+))?$`)

func (c *UrlContext) BuildBlog(page int) string {
	defer CatchPanic()
	if page < 1 {
		panic(oops.New(nil, "Invalid blog page (%d), must be >= 1", page))
	}
	path := "/blog"

	if page > 1 {
		path += "/" + strconv.Itoa(page)
	}

	return c.Url(path, nil)
}

var RegexBlogThread = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)(-([^/]+))?$`)

func (c *UrlContext) BuildBlogThread(threadId int, title string) string {
	defer CatchPanic()
	builder := buildBlogThreadPath(threadId, title)
	return c.Url(builder.String(), nil)
}

func (c *UrlContext) BuildBlogThreadWithPostHash(threadId int, title string, postId int) string {
	defer CatchPanic()
	builder := buildBlogThreadPath(threadId, title)
	return c.UrlWithFragment(builder.String(), nil, strconv.Itoa(postId))
}

var RegexBlogNewThread = regexp.MustCompile(`^/blog/new$`)

func (c *UrlContext) BuildBlogNewThread() string {
	defer CatchPanic()
	return c.Url("/blog/new", nil)
}

var RegexBlogPost = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)/e/(?P<postid>\d+)$`)

func (c *UrlContext) BuildBlogPost(threadId int, postId int) string {
	defer CatchPanic()
	builder := buildBlogPostPath(threadId, postId)
	return c.Url(builder.String(), nil)
}

var RegexBlogPostDelete = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)/e/(?P<postid>\d+)/delete$`)

func (c *UrlContext) BuildBlogPostDelete(threadId int, postId int) string {
	defer CatchPanic()
	builder := buildBlogPostPath(threadId, postId)
	builder.WriteString("/delete")
	return c.Url(builder.String(), nil)
}

var RegexBlogPostEdit = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)/e/(?P<postid>\d+)/edit$`)

func (c *UrlContext) BuildBlogPostEdit(threadId int, postId int) string {
	defer CatchPanic()
	builder := buildBlogPostPath(threadId, postId)
	builder.WriteString("/edit")
	return c.Url(builder.String(), nil)
}

var RegexBlogPostReply = regexp.MustCompile(`^/blog/p/(?P<threadid>\d+)/e/(?P<postid>\d+)/reply$`)

func (c *UrlContext) BuildBlogPostReply(threadId int, postId int) string {
	defer CatchPanic()
	builder := buildBlogPostPath(threadId, postId)
	builder.WriteString("/reply")
	return c.Url(builder.String(), nil)
}

/*
* Library (old)
 */

var RegexLibraryAny = regexp.MustCompile(`^/library`)

/*
* Episode Guide
 */

var RegexEpisodeList = regexp.MustCompile(`^/episode(/(?P<topic>[^/]+))?$`)

func (c *UrlContext) BuildEpisodeList(topic string) string {
	defer CatchPanic()

	var builder strings.Builder
	builder.WriteString("/episode")
	if topic != "" {
		builder.WriteString("/")
		builder.WriteString(topic)
	}
	return c.Url(builder.String(), nil)
}

var RegexEpisode = regexp.MustCompile(`^/episode/(?P<topic>[^/]+)/(?P<episode>[^/]+)$`)

func (c *UrlContext) BuildEpisode(topic string, episode string) string {
	defer CatchPanic()
	return c.Url(fmt.Sprintf("/episode/%s/%s", topic, episode), nil)
}

var RegexCineraIndex = regexp.MustCompile(`^/(?P<topic>[^/]+).index$`)

func (c *UrlContext) BuildCineraIndex(topic string) string {
	defer CatchPanic()
	return c.Url(fmt.Sprintf("/%s.index", topic), nil)
}

/*
* Discord OAuth
 */

var RegexDiscordOAuthCallback = regexp.MustCompile("^/_discord_callback$")

func BuildDiscordOAuthCallback() string {
	return Url("/_discord_callback", nil)
}

var RegexDiscordUnlink = regexp.MustCompile("^/_discord_unlink$")

func BuildDiscordUnlink() string {
	return Url("/_discord_unlink", nil)
}

var RegexDiscordShowcaseBacklog = regexp.MustCompile("^/discord_showcase_backlog$")

func BuildDiscordShowcaseBacklog() string {
	return Url("/discord_showcase_backlog", nil)
}

var RegexDiscordBotDebugPage = regexp.MustCompile("^/discord_bot_debug$")

func BuildDiscordBotDebugPage() string {
	return Url("/discord_bot_debug", nil)
}

/*
* API
 */

var RegexAPICheckUsername = regexp.MustCompile("^/api/check_username$")

func BuildAPICheckUsername() string {
	return Url("/api/check_username", nil)
}

var RegexAPINewsletterSignup = regexp.MustCompile("^/api/newsletter_signup$")

func BuildAPINewsletterSignup() string {
	return Url("/api/newsletter_signup", nil)
}

/*
* Twitch stuff
 */

var RegexTwitchEventSubCallback = regexp.MustCompile("^/twitch_eventsub$")

func BuildTwitchEventSubCallback() string {
	return Url("/twitch_eventsub", nil)
}

var RegexTwitchDebugPage = regexp.MustCompile("^/twitch_debug$")

/*
* Following
 */

var RegexFollowingTest = regexp.MustCompile("^/following$")

func BuildFollowingTest() string {
	return Url("/following", nil)
}

var RegexFollowUser = regexp.MustCompile("^/follow/user$")

func BuildFollowUser() string {
	return Url("/follow/user", nil)
}

var RegexFollowProject = regexp.MustCompile("^/follow/project$")

func BuildFollowProject() string {
	return Url("/follow/project", nil)
}

/*
* Perf
 */

var RegexPerfmon = regexp.MustCompile("^/perfmon$")

func BuildPerfmon() string {
	return Url("/perfmon", nil)
}

/*
* User assets
 */

var RegexAssetUpload = regexp.MustCompile("^/upload_asset$")

// NOTE(asaf): Providing the projectSlug avoids any CORS problems.
func (c *UrlContext) BuildAssetUpload() string {
	return c.Url("/upload_asset", nil)
}

/*
* Assets
 */

var RegexMarkdownWorkerJS = regexp.MustCompile("^/assets/markdown_worker.js$")

func BuildMarkdownWorkerJS() string {
	defer CatchPanic()
	return Url("/assets/markdown_worker.js", nil)
}

var RegexS3Asset *regexp.Regexp

func BuildS3Asset(s3key string) string {
	defer CatchPanic()
	res := fmt.Sprintf("%s%s", S3BaseUrl, s3key)
	return res
}

var RegexEsBuild = regexp.MustCompile("^/esbuild$")

func BuildEsBuild() string {
	defer CatchPanic()
	return Url("/esbuild", nil)
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
		query = []Q{{"v", cacheBustVersion}}
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
	if filepath == "" {
		return ""
	}

	filepath = strings.Trim(filepath, "/")
	return BuildPublic(fmt.Sprintf("media/%s", filepath), false)
}

/*
* Redirects
 */

var RegexUnwind = regexp.MustCompile(`^/unwind$`)

/*
* Other
 */

var RegexForumMarkRead = regexp.MustCompile(`^/markread/(?P<sfid>\d+)$`)

// NOTE(asaf): subforumId == 0 means ALL SUBFORUMS
func (c *UrlContext) BuildForumMarkRead(subforumId int) string {
	defer CatchPanic()
	if subforumId < 0 {
		panic(oops.New(nil, "Invalid subforum ID (%d), must be >= 0", subforumId))
	}

	var builder strings.Builder
	builder.WriteString("/markread/")
	builder.WriteString(strconv.Itoa(subforumId))

	return c.Url(builder.String(), nil)
}

var RegexCatchAll = regexp.MustCompile("^")

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

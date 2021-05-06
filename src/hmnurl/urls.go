package hmnurl

import (
	"regexp"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/oops"
)

var RegexHomepage = regexp.MustCompile("^/$")

func BuildHomepage() string {
	return Url("/", nil)
}

var RegexLogin = regexp.MustCompile("^/login$")

func BuildLogin() string {
	return Url("/login", nil)
}

var RegexLogout = regexp.MustCompile("^/logout$")

func BuildLogout() string {
	return Url("/logout", nil)
}

var RegexManifesto = regexp.MustCompile("^/manifesto$")

func BuildManifesto() string {
	return Url("/manifesto", nil)
}

var RegexAbout = regexp.MustCompile("^/about$")

func BuildAbout() string {
	return Url("/about", nil)
}

var RegexCodeOfConduct = regexp.MustCompile("^/code-of-conduct$")

func BuildCodeOfConduct() string {
	return Url("/code-of-conduct", nil)
}

var RegexCommunicationGuidelines = regexp.MustCompile("^/communication-guidelines$")

func BuildCommunicationGuidelines() string {
	return Url("/communication-guidelines", nil)
}

var RegexContactPage = regexp.MustCompile("^/contact$")

func BuildContactPage() string {
	return Url("/contact", nil)
}

var RegexMonthlyUpdatePolicy = regexp.MustCompile("^/monthly-update-policy$")

func BuildMonthlyUpdatePolicy() string {
	return Url("/monthly-update-policy", nil)
}

var RegexProjectSubmissionGuidelines = regexp.MustCompile("^/project-guidelines$")

func BuildProjectSubmissionGuidelines() string {
	return Url("/project-guidelines", nil)
}

var RegexFeed = regexp.MustCompile(`^/feed(/(?P<page>.+)?)?$`)

func BuildFeed() string {
	return Url("/feed", nil)
}

func BuildFeedWithPage(page int) string {
	if page < 1 {
		panic(oops.New(nil, "Invalid feed page (%d), must be >= 1", page))
	}
	if page == 1 {
		return BuildFeed()
	}
	return Url("/feed/"+strconv.Itoa(page), nil)
}

var RegexForumThread = regexp.MustCompile(`^/(?P<cats>forums(/[^\d]+?)*)/t/(?P<threadid>\d+)(/(?P<page>\d+))?$`)

func BuildForumThread(projectSlug string, subforums []string, threadId int, page int) string {
	if page < 1 {
		panic(oops.New(nil, "Invalid forum thread page (%d), must be >= 1", page))
	}

	var builder strings.Builder
	builder.WriteString("/forums")
	for _, subforum := range subforums {
		subforum = strings.TrimSpace(subforum)
		if strings.Contains(subforum, "/") {
			panic(oops.New(nil, "Tried building forum thread url with / in subforum name"))
		}
		if len(subforum) == 0 {
			panic(oops.New(nil, "Tried building forum thread url with blank subforum"))
		}
		builder.WriteRune('/')
		builder.WriteString(subforum)
	}
	builder.WriteString("/t/")
	builder.WriteString(strconv.Itoa(threadId))
	if page > 1 {
		builder.WriteRune('/')
		builder.WriteString(strconv.Itoa(page))
	}

	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexForumCategory = regexp.MustCompile(`^/(?P<cats>forums(/[^\d]+?)*)(/(?P<page>\d+))?$`)

func BuildForumCategory(projectSlug string, subforums []string, page int) string {
	if page < 1 {
		panic(oops.New(nil, "Invalid forum thread page (%d), must be >= 1", page))
	}

	var builder strings.Builder
	builder.WriteString("/forums")
	for _, subforum := range subforums {
		subforum = strings.TrimSpace(subforum)
		if strings.Contains(subforum, "/") {
			panic(oops.New(nil, "Tried building forum thread url with / in subforum name"))
		}
		if len(subforum) == 0 {
			panic(oops.New(nil, "Tried building forum thread url with blank subforum"))
		}
		builder.WriteRune('/')
		builder.WriteString(subforum)
	}
	if page > 1 {
		builder.WriteRune('/')
		builder.WriteString(strconv.Itoa(page))
	}

	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexForumPost = regexp.MustCompile(``) // TODO(asaf): Complete this and test it

func BuildForumPost(projectSlug string, subforums []string, threadId int, postId int) string {
	var builder strings.Builder
	builder.WriteString("/forums")
	for _, subforum := range subforums {
		subforum = strings.TrimSpace(subforum)
		if strings.Contains(subforum, "/") {
			panic(oops.New(nil, "Tried building forum thread url with / in subforum name"))
		}
		if len(subforum) == 0 {
			panic(oops.New(nil, "Tried building forum thread url with blank subforum"))
		}
		builder.WriteRune('/')
		builder.WriteString(subforum)
	}
	builder.WriteString("/t/")
	builder.WriteString(strconv.Itoa(threadId))
	builder.WriteString("/p/")
	builder.WriteString(strconv.Itoa(postId))

	return ProjectUrl(builder.String(), nil, projectSlug)
}

var RegexForumPostDelete = regexp.MustCompile(``) // TODO

func BuildForumPostDelete(projectSlug string, subforums []string, threadId int, postId int) string {
	return BuildForumPost(projectSlug, subforums, threadId, postId) + "/delete"
}

var RegexForumPostEdit = regexp.MustCompile(``) // TODO

func BuildForumPostEdit(projectSlug string, subforums []string, threadId int, postId int) string {
	return BuildForumPost(projectSlug, subforums, threadId, postId) + "/edit"
}

var RegexForumPostReply = regexp.MustCompile(``) // TODO(asaf): Ha ha! I, Ben, have played a trick on you, and forced you to do this regex as well!

// TODO: It's kinda weird that we have "replies" to a specific post. That's not how the data works. I guess this just affects what you see as the "post you're replying to" on the post composer page?
func BuildForumPostReply(projectSlug string, subforums []string, threadId int, postId int) string {
	return BuildForumPost(projectSlug, subforums, threadId, postId) + "/reply"
}

var RegexForumPostQuote = regexp.MustCompile(``) // TODO

func BuildForumPostQuote(projectSlug string, subforums []string, threadId int, postId int) string {
	return BuildForumPost(projectSlug, subforums, threadId, postId) + "/quote"
}

var RegexProjectCSS = regexp.MustCompile("^/assets/project.css$")

func BuildProjectCSS(color string) string {
	return Url("/assets/project.css", []Q{{"color", color}})
}

var RegexPublic = regexp.MustCompile("^/public/.+$")

func BuildPublic(filepath string) string {
	filepath = strings.Trim(filepath, "/")
	if len(strings.TrimSpace(filepath)) == 0 {
		panic(oops.New(nil, "Attempted to build a /public url with no path"))
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
	return Url(builder.String(), nil)
}

var RegexCatchAll = regexp.MustCompile("")

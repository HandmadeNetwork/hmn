package website

import (
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

// This will skip the common path prefix for fishbowl files.
// We unfortunately need to do this because we want to use http.FileServer,
// but _that_ needs an http.FS, but _that_ needs an fs.FS...
var fishbowlFS = utils.Must1(fs.Sub(templates.FishbowlFS, "src/fishbowls"))
var fishbowlHTTPFS = http.StripPrefix("/fishbowl", http.FileServer(http.FS(fishbowlFS)))

type fishbowlInfo struct {
	Slug               string
	Title, Description string // The description is used for OpenGraph, so it must be plain text, no HTML.
	Month              time.Month
	Year               int
	ContentsPath       string
}

var fishbowls = [...]fishbowlInfo{
	{
		Slug:        "internet-os",
		Title:       "The future of operating systems in an Internet world",
		Description: `Despite the web's technical problems, it dominates software development today, largely due to its cross-platform support and ease of distribution. At the same time, our discussions about the future of programming tend to involve new "operating systems", but those discussions rarely take the Internet into account. What could future operating systems look like in a world defined by the Internet?`,
		Month:       time.May, Year: 2020,
		ContentsPath: "internet-os/internet-os.html",
	},
	{
		Slug:        "metaprogramming",
		Title:       "Compile-time introspection and metaprogramming",
		Description: `Thanks to new languages like Zig and Jai, compile-time execution and metaprogramming are a popular topic of discussion in the community. This fishbowl explores metaprogramming in more detail, and discusses to what extent it is actually necessary, or just a waste of time.`,
		Month:       time.June, Year: 2020,
		ContentsPath: "metaprogramming/metaprogramming.html",
	},
	{
		Slug:        "lisp-jam",
		Title:       "Lessons from the Lisp Jam",
		Description: `In the summer of 2020 we held a Lisp jam, where many community members made exploratory Lisp-inspired projects. We held this fishbowl as a recap, as a time for the participants to share what they learned and explore how those lessons relate to our day-to-day programming.`,
		Month:       time.August, Year: 2020,
		ContentsPath: "lisp-jam/lisp-jam.html",
	},
	{
		Slug:        "parallel-programming",
		Title:       "Approaches to parallel programming",
		Description: `A discussion of many aspects of parallelism and concurrency in programming, and the pros and cons of different programming methodologies.`,
		Month:       time.November, Year: 2020,
		ContentsPath: "parallel-programming/parallel-programming.html",
	},
	{
		Slug:        "skimming",
		Title:       "Code skimmability as the root cause for bad code structure decisions", // real snappy, this one
		Description: `Programmers tend to care a lot about "readability". This usually means having small classes, small functions, small files. This code might be "readable" at a glance, but this doesn't really help you understand the program—it's just "skimmable". How can we think about "readability" in a more productive way?`,
		Month:       time.January, Year: 2021,
		ContentsPath: "skimming/skimming.html",
	},
	{
		Slug:        "config",
		Title:       "How to design to avoid configuration",
		Description: `Configuration sucks. How can we avoid it, while still making software that supports a wide range of behaviors? What is the essence of "configuration", and how can we identify it? How can we identify what is "bad config", and design our software to avoid it?`,
		Month:       time.March, Year: 2021,
		ContentsPath: "config/config.html",
	},
	{
		Slug:        "simplicity-performance",
		Title:       "The relationship of simplicity and performance",
		Description: "In the community, we talk a lot about performance. We also talk a lot about having simple code—and the two feel somewhat intertwined. What relationship is there between simplicity and performance? Are there better ways to reason about \"simplicity\" with this in mind?",
		Month:       time.May, Year: 2021,
		ContentsPath: "simplicity-performance/simplicity-performance.html",
	},
	{
		Slug:        "teaching-software",
		Title:       "How software development is taught",
		Description: "The Handmade Network exists because we are unhappy with the software status quo. To a large extent, this is because of how software development is taught. What are the good parts of software education today, what are the flaws, and how might we change things to improve the state of software?",
		Month:       time.June, Year: 2021,
		ContentsPath: "teaching-software/teaching-software.html",
	},
	{
		Slug:        "flexible-software",
		Title:       "How to design flexible software",
		Description: "We previously held a fishbowl about how to design to avoid configuration. But when you can't avoid configuration, how do you do it well? And if we want our software to be flexible, what other options do we have besides configuration? What other ways are there to make software flexible?",
		Month:       time.December, Year: 2021,
		ContentsPath: "flexible-software/flexible-software.html",
	},
	{
		Slug:        "oop",
		Title:       "What, if anything, is OOP?",
		Description: "Is object-oriented programming bad? Is it good? What even is it, anyway? This fishbowl explores OOP more carefully—what is the essence of it, what are the good parts, why did it take over the world, and why do we criticize it so much?",
		Month:       time.May, Year: 2022,
		ContentsPath: "oop/OOP.html",
	},
	{
		Slug:        "libraries",
		Title:       "When do libraries go sour?",
		Description: "The Handmade community is often opposed to using libraries. But let's get more specific about why that can be, and whether that's reasonable. What do we look for in a library? When do libraries go sour? How do we evaluate libraries before using them? How can the libraries we make avoid these problems?",
		Month:       time.July, Year: 2022,
		ContentsPath: "libraries/libraries.html",
	},
	{
		Slug:        "entrepreneurship",
		Title:       "Entrepreneurship and the Handmade ethos",
		Description: "What does it look like to turn a Handmade project into a sustainable business? How can the Handmade ethos set our software apart from its competitors? And how might we sustain the development of important software, if we're not sure how to sell it?",
		Month:       time.October, Year: 2022,
		ContentsPath: "entrepreneurship/entrepreneurship.html",
	},
	{
		Slug:        "testing",
		Title:       "What even is testing?",
		Description: "Everybody knows testing is important, but the software industry is overrun by terrible testing practices. Because of this, there has often been a negative sentiment against testing in the Handmade community. This fishbowl explores the kinds of testing the community has found most effective, the costs of testing, and the actual purpose behind testing techniques.",
		Month:       time.May, Year: 2023,
		ContentsPath: "testing/testing.html",
	},
}

func FishbowlIndex(c *RequestContext) ResponseData {
	type fishbowlTmpl struct {
		Fishbowl fishbowlInfo
		Url      string
		Valid    bool

		date time.Time
	}
	type tmpl struct {
		templates.BaseData
		Fishbowls []fishbowlTmpl
	}

	tmplData := tmpl{
		BaseData: getBaseData(c, "Fishbowls", []templates.Breadcrumb{
			{Name: "Fishbowls", Url: hmnurl.BuildFishbowlIndex()},
		}),
	}

	var fishbowlTmpls []fishbowlTmpl
	for _, f := range fishbowls {
		fishbowlTmpls = append(fishbowlTmpls, fishbowlTmpl{
			Fishbowl: f,
			Url:      hmnurl.BuildFishbowl(f.Slug),
			Valid:    f.ContentsPath != "",

			date: time.Date(f.Year, f.Month, 0, 0, 0, 0, 0, time.UTC),
		})
	}
	sort.Slice(fishbowlTmpls, func(i, j int) bool {
		return fishbowlTmpls[j].date.Before(fishbowlTmpls[i].date) // reverse
	})
	tmplData.Fishbowls = fishbowlTmpls

	var res ResponseData
	err := res.WriteTemplate("fishbowl_index.html", tmplData, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render fishbowl index page"))
	}
	return res
}

func Fishbowl(c *RequestContext) ResponseData {
	slug := c.PathParams["slug"]
	var info fishbowlInfo

	// Only serve up valid fishbowls (with content)
	exists := false
	for _, fishbowl := range fishbowls {
		if fishbowl.Slug == slug && fishbowl.ContentsPath != "" {
			exists = true
			info = fishbowl
		}
	}
	if !exists {
		return FourOhFour(c)
	}

	// Ensure trailing slash (it matters for relative URLs in the HTML)
	if !strings.HasSuffix(c.URL().Path, "/") {
		return c.Redirect(c.URL().Path+"/", http.StatusFound)
	}

	type FishbowlData struct {
		templates.BaseData
		Slug     string
		Info     fishbowlInfo
		Contents template.HTML
	}

	contentsFile := utils.Must1(fishbowlFS.Open(info.ContentsPath))
	contents := string(utils.Must1(io.ReadAll(contentsFile)))
	contents, err := linkifyDiscordContent(c, c.Conn, contents)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to linkify fishbowl content"))
	}

	tmpl := FishbowlData{
		BaseData: getBaseData(c, info.Title, []templates.Breadcrumb{
			{Name: "Fishbowls", Url: hmnurl.BuildFishbowlIndex()},
			{Name: info.Title, Url: hmnurl.BuildFishbowl(slug)},
		}),
		Slug:     slug,
		Info:     info,
		Contents: template.HTML(contents),
	}
	tmpl.BaseData.OpenGraphItems = append(tmpl.BaseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:description",
		Value:    info.Description,
	})

	var res ResponseData
	err = res.WriteTemplate("fishbowl.html", tmpl, c.Perf)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to render fishbowl index page"))
	}
	return res
}

func FishbowlFiles(c *RequestContext) ResponseData {
	var res ResponseData
	fishbowlHTTPFS.ServeHTTP(&res, c.Req)
	addCORSHeaders(c, &res)
	return res
}

var reFishbowlDiscordUserId = regexp.MustCompile(`data-user-id="(\d+)"`)
var reFishbowlDiscordAuthorHeader = regexp.MustCompile(`(?s:(<div class="chatlog__message">.*?)(<img class="chatlog__avatar".*?>)(.*?<span class="chatlog__author".*?data-user-id="(\d+)".*?>)(.*?)(</span>))`)

func linkifyDiscordContent(c *RequestContext, dbConn db.ConnOrTx, content string) (string, error) {
	discordUserIdSet := make(map[string]struct{})
	userIdMatches := reFishbowlDiscordUserId.FindAllStringSubmatch(content, -1)
	for _, m := range userIdMatches {
		discordUserIdSet[m[1]] = struct{}{}
	}
	discordUserIds := make([]string, 0, len(discordUserIdSet))
	for id := range discordUserIdSet {
		discordUserIds = append(discordUserIds, id)
	}

	hmnUsers, err := hmndata.FetchUsers(c, dbConn, c.CurrentUser, hmndata.UsersQuery{
		DiscordUserIDs: discordUserIds,
	})
	if err != nil {
		return "", err
	}

	return reFishbowlDiscordAuthorHeader.ReplaceAllStringFunc(content, func(s string) string {
		m := reFishbowlDiscordAuthorHeader.FindStringSubmatch(s)
		discordUserID := m[4]

		var matchingUser *models.User
		for _, u := range hmnUsers {
			if u.DiscordUser.UserID == discordUserID {
				matchingUser = u
				break
			}
		}

		if matchingUser == nil {
			return s
		} else {
			link := fmt.Sprintf(`<a href="%s" target="_blank">`, hmnurl.BuildUserProfile(matchingUser.Username))
			return m[1] + link + m[2] + "</a>" + m[3] + link + m[5] + "</a>" + m[6]
		}
	}), nil
}

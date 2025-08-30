package website

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

const JamRecentWindow = 14 * 24 * time.Hour
const JamBannerGraceBefore = 30 * 24 * time.Hour
const JamBannerGraceAfter = 14 * 24 * time.Hour

func JamCurrentTime(c *RequestContext, ev hmndata.Event) time.Time {
	t := time.Now()
	testTime := c.Req.URL.Query().Get("testtime")
	if testTime == "pre" {
		t = ev.StartTime.Add(-100 * time.Hour)
	} else if testTime == "during" {
		t = ev.StartTime.Add(10 * time.Minute)
	} else if testTime == "post" {
		t = ev.EndTime.Add(100 * time.Hour)
	} else if days, err := strconv.Atoi(testTime); err == nil {
		t = ev.StartTime.Add(time.Duration(days) * 24 * time.Hour)
	} else if parsed, err := time.Parse(time.RFC3339, testTime); err == nil {
		t = parsed
	}
	return t
}

func JamsIndex(c *RequestContext) ResponseData {
	var res ResponseData

	type TemplateData struct {
		templates.BaseData

		LispJamUrl  string
		WRJ2021Url  string
		WRJ2022Url  string
		VJ2023Url   string
		WRJ2023Url  string
		LJ2024Url   string
		VJ2024Url   string
		WRJ2024Url  string
		XRay2025Url string
	}

	res.MustWriteTemplate("jams_index.html", TemplateData{
		BaseData: getBaseData(c, "Jams", nil),

		LispJamUrl:  hmnurl.BuildFishbowl("lisp-jam"),
		WRJ2021Url:  hmnurl.BuildJamIndex2021(),
		WRJ2022Url:  hmnurl.BuildJamIndex2022(),
		VJ2023Url:   hmnurl.BuildJamIndex2023_Visibility(),
		WRJ2023Url:  hmnurl.BuildJamIndex2023(),
		LJ2024Url:   hmnurl.BuildJamIndex2024_Learning(),
		VJ2024Url:   hmnurl.BuildJamIndex2024_Visibility(),
		WRJ2024Url:  hmnurl.BuildJamGenericIndex(hmndata.WRJ2024.UrlSlug),
		XRay2025Url: hmnurl.BuildJamGenericIndex(hmndata.XRay2025.UrlSlug),
	}, c.Perf)
	return res
}

type JamAssets struct {
	TwitterCard string
	Logo        string
}

type JamGenericTemplateData struct {
	templates.BaseData
	Assets JamAssets

	Timespans                  hmndata.EventTimespans
	StartTimeUnix, EndTimeUnix int64
	JamUrl                     string
	JamFeedUrl                 string
	NewProjectUrl              string
	GuidelinesUrl              string
	TwitchEmbedUrl             string
	RecapStreamEmbedUrl        string
	SubmittedProject           *templates.Project
	JamProjects                []templates.Project
	TimelineItems              []templates.TimelineItem
	ShortFeed                  bool
}

func getJamAssets(jam hmndata.Jam) JamAssets {
	return JamAssets{
		TwitterCard: fmt.Sprintf("jams/%s/TwitterCard.png", jam.UrlSlug),
		Logo:        fmt.Sprintf("jams/%s/logo.svg", jam.UrlSlug),
	}
}

func getJamGenericTemplateData(c *RequestContext, jam hmndata.Jam, baseData templates.BaseData, numTimelineItems int) (JamGenericTemplateData, error) {
	now := JamCurrentTime(c, jam.Event)

	assets := getJamAssets(jam)

	opengraph := []templates.OpenGraphItem{
		{Property: "og:title", Value: jam.Name},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic(assets.TwitterCard, true)},
		{Property: "og:description", Value: jam.Description},
		{Property: "og:url", Value: hmnurl.BuildJamGenericIndex(jam.UrlSlug)},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic(assets.TwitterCard, true)},
	}

	baseData.OpenGraphItems = opengraph
	baseData.BodyClasses = append(baseData.BodyClasses, "header-transparent")
	baseData.ForceDark = jam.ForceDark
	baseData.Header.SuppressBanners = true

	var submittedProject *templates.Project
	if c.CurrentUser != nil {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{jam.Slug},
			Limit:    1,
		})
		if err != nil {
			return JamGenericTemplateData{}, oops.New(err, "failed to fetch current user's jam project")
		}
		if len(projects) > 0 {
			submittedProject = utils.P(templates.ProjectAndStuffToTemplate(&projects[0]))
		}
	}

	var newProjectUrl string
	if jam.Event.WithinGrace(JamCurrentTime(c, jam.Event), hmndata.JamProjectCreateGracePeriod, 0) {
		newProjectUrl = hmnurl.BuildProjectNewJam()
	}

	pageProjects := []templates.Project{}
	timelineItems := []templates.TimelineItem{}

	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{jam.Slug},
	})
	if err != nil {
		return JamGenericTemplateData{}, oops.New(err, "failed to fetch jam projects")
	}

	projectIDs := make([]int, 0, len(jamProjects))
	pageProjects = make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
		projectIDs = append(projectIDs, p.Project.ID)
	}

	if len(jamProjects) > 0 {
		timelineItems, err = FetchTimeline(c, c.Conn, c.CurrentUser, nil, hmndata.TimelineQuery{
			ProjectIDs: projectIDs,
			SkipPosts:  true,
			Limit:      numTimelineItems,
		})
	}

	twitchEmbedUrl := getTwitchEmbedUrl(c)

	templateData := JamGenericTemplateData{
		BaseData: baseData,
		Assets:   assets,

		StartTimeUnix: jam.StartTime.Unix(),
		EndTimeUnix:   jam.EndTime.Unix(),
		Timespans:     hmndata.CalcTimespans(jam.Event, now),

		JamUrl:              hmnurl.BuildJamGenericIndex(jam.UrlSlug),
		JamFeedUrl:          hmnurl.BuildJamGenericFeed(jam.UrlSlug),
		NewProjectUrl:       newProjectUrl,
		GuidelinesUrl:       hmnurl.BuildJamGenericGuidelines(jam.UrlSlug),
		TwitchEmbedUrl:      twitchEmbedUrl,
		RecapStreamEmbedUrl: jam.RecapStreamEmbedUrl,
		SubmittedProject:    submittedProject,
		JamProjects:         pageProjects,
		TimelineItems:       timelineItems,
	}

	return templateData, nil
}

func findJamByUrlSlug(urlSlug string) (hmndata.Jam, bool) {
	for _, j := range hmndata.AllJams {
		if strings.ToLower(j.UrlSlug) == urlSlug {
			return j, true
		}
	}

	return hmndata.Jam{}, false
}

func JamGenericIndex(c *RequestContext) ResponseData {
	var res ResponseData

	urlSlug := strings.ToLower(c.PathParams["urlslug"])

	jam, found := findJamByUrlSlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	templateData, err := getJamGenericTemplateData(c, jam, getBaseData(c, jam.Name, nil), 10)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	templateName := fmt.Sprintf("jam_%s_index.html", jam.TemplateName)

	res.MustWriteTemplate(templateName, templateData, c.Perf)
	return res
}

func JamGenericFeed(c *RequestContext) ResponseData {
	var res ResponseData

	urlSlug := strings.ToLower(c.PathParams["urlslug"])

	jam, found := findJamByUrlSlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	templateData, err := getJamGenericTemplateData(c, jam, getBaseData(c, jam.Name, nil), 10)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	templateName := fmt.Sprintf("jam_%s_feed.html", jam.TemplateName)

	res.MustWriteTemplate(templateName, templateData, c.Perf)
	return res
}

func JamGenericGuidelines(c *RequestContext) ResponseData {
	var res ResponseData

	urlSlug := strings.ToLower(c.PathParams["urlslug"])

	jam, found := findJamByUrlSlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	templateData, err := getJamGenericTemplateData(c, jam, getBaseData(c, jam.Name, nil), 10)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	templateName := fmt.Sprintf("jam_%s_guidelines.html", jam.TemplateName)

	res.MustWriteTemplate(templateName, templateData, c.Perf)
	return res
}

type JamBaseDataVJ2024 struct {
	Timespans                  hmndata.EventTimespans
	StartTimeUnix, EndTimeUnix int64
	JamUrl                     string
	JamFeedUrl                 string
	NewProjectUrl              string
	GuidelinesUrl              string
	SubmittedProject           *templates.Project
}

type JamPageDataVJ2024 struct {
	templates.BaseData
	JamBaseDataVJ2024

	TwitchEmbedUrl string
	JamProjects    []templates.Project
	TimelineItems  []templates.TimelineItem
	ShortFeed      bool
}

func JamIndex2024_Visibility(c *RequestContext) ResponseData {
	var res ResponseData

	jam := hmndata.VJ2024
	now := JamCurrentTime(c, jam.Event)

	jamBaseData, err := getVJ2024BaseData(c, now)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	baseData := getBaseData(c, jam.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2024/opengraph.png", true)},
		{Property: "og:description", Value: "See things in a new way. July 19 - 21."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2024_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2024/TwitterCard.png", true)},
	}
	baseData.BodyClasses = append(baseData.BodyClasses, "header-transparent")
	baseData.Header.SuppressBanners = true

	pageProjects := []templates.Project{}
	timelineItems := []templates.TimelineItem{}
	twitchEmbedUrl := ""

	if jamBaseData.Timespans.AfterStart {
		jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			JamSlugs: []string{jam.Slug},
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
		}

		projectIDs := make([]int, 0, len(jamProjects))
		pageProjects = make([]templates.Project, 0, len(jamProjects))
		for _, p := range jamProjects {
			pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
			projectIDs = append(projectIDs, p.Project.ID)
		}

		if len(jamProjects) > 0 {
			timelineItems, err = FetchTimeline(c, c.Conn, c.CurrentUser, nil, hmndata.TimelineQuery{
				ProjectIDs: projectIDs,
				SkipPosts:  true,
				Limit:      10,
			})
		}

		twitchEmbedUrl = getTwitchEmbedUrl(c)
	}

	res.MustWriteTemplate("jam_2024_vj_index.html", JamPageDataVJ2024{
		BaseData:          baseData,
		JamBaseDataVJ2024: jamBaseData,
		JamProjects:       pageProjects,
		TimelineItems:     timelineItems,
		ShortFeed:         true,
		TwitchEmbedUrl:    twitchEmbedUrl,
	}, c.Perf)
	return res
}

func JamFeed2024_Visibility(c *RequestContext) ResponseData {
	var res ResponseData

	jam := hmndata.VJ2024
	now := JamCurrentTime(c, jam.Event)

	jamBaseData, err := getVJ2024BaseData(c, now)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	baseData := getBaseData(c, jam.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2024/opengraph.png", true)},
		{Property: "og:description", Value: "See things in a new way. July 19 - 21."},
		{Property: "og:url", Value: hmnurl.BuildJamFeed2024_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2024/TwitterCard.png", true)},
	}
	baseData.BodyClasses = append(baseData.BodyClasses, "header-transparent")
	baseData.Header.SuppressBanners = true

	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{jam.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	projectIDs := make([]int, 0, len(jamProjects))
	for _, p := range jamProjects {
		projectIDs = append(projectIDs, p.Project.ID)
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
	}

	timelineItems := []templates.TimelineItem{}

	if len(projectIDs) > 0 {
		timelineItems, err = FetchTimeline(c, c.Conn, c.CurrentUser, nil, hmndata.TimelineQuery{
			ProjectIDs: projectIDs,
			SkipPosts:  true,
		})
	}

	res.MustWriteTemplate("jam_2024_vj_feed.html", JamPageDataVJ2024{
		BaseData:          baseData,
		JamBaseDataVJ2024: jamBaseData,
		JamProjects:       pageProjects,
		TimelineItems:     timelineItems,
	}, c.Perf)
	return res
}

func JamGuidelines2024_Visibility(c *RequestContext) ResponseData {
	var res ResponseData

	jam := hmndata.VJ2024
	now := JamCurrentTime(c, jam.Event)

	jamBaseData, err := getVJ2024BaseData(c, now)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	baseData := getBaseData(c, jam.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2024/opengraph.png", true)},
		{Property: "og:description", Value: "See things in a new way. July 19 - 21."},
		{Property: "og:url", Value: hmnurl.BuildJamGuidelines2024_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2024/TwitterCard.png", true)},
	}
	baseData.BodyClasses = append(baseData.BodyClasses, "header-transparent")
	baseData.Header.SuppressBanners = true

	res.MustWriteTemplate("jam_2024_vj_guidelines.html", JamPageDataVJ2024{
		BaseData:          baseData,
		JamBaseDataVJ2024: jamBaseData,
	}, c.Perf)
	return res
}

func getVJ2024BaseData(c *RequestContext, now time.Time) (JamBaseDataVJ2024, error) {
	jam := hmndata.VJ2024

	var submittedProject *templates.Project
	if c.CurrentUser != nil {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{jam.Slug},
			Limit:    1,
		})
		if err != nil {
			return JamBaseDataVJ2024{}, oops.New(err, "failed to fetch jam projects for current user")
		}
		if len(projects) > 0 {
			submittedProject = utils.P(templates.ProjectAndStuffToTemplate(&projects[0]))
		}
	}

	return JamBaseDataVJ2024{
		StartTimeUnix: jam.StartTime.Unix(),
		EndTimeUnix:   jam.EndTime.Unix(),
		Timespans:     hmndata.CalcTimespans(jam.Event, now),

		JamUrl:           hmnurl.BuildJamIndex2024_Visibility(),
		JamFeedUrl:       hmnurl.BuildJamFeed2024_Visibility(),
		NewProjectUrl:    hmnurl.BuildProjectNewJam(),
		GuidelinesUrl:    hmnurl.BuildJamGuidelines2024_Visibility(),
		SubmittedProject: submittedProject,
	}, nil
}

func JamIndex2024_Learning(c *RequestContext) ResponseData {
	var res ResponseData

	baseData := getBaseData(c, hmndata.LJ2024.Name, nil)
	baseData.OpenGraphItems = opengraphLJ2024

	jamBaseData, err := getLJ2024BaseData(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	feedData, err := getLJ2024FeedData(c, 5)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	type JamPageData struct {
		templates.BaseData
		JamBaseDataLJ2024
		TwitchEmbedUrl string

		Projects      JamProjectDataLJ2024
		TimelineItems []templates.TimelineItem
	}

	tmpl := JamPageData{
		BaseData:          baseData,
		JamBaseDataLJ2024: jamBaseData,
		TwitchEmbedUrl:    getTwitchEmbedUrl(c),

		Projects:      feedData.Projects,
		TimelineItems: feedData.TimelineItems,
	}

	res.MustWriteTemplate("jam_2024_lj_index.html", tmpl, c.Perf)
	return res
}

func JamFeed2024_Learning(c *RequestContext) ResponseData {
	baseData := getBaseData(c, hmndata.LJ2024.Name, nil)
	baseData.OpenGraphItems = opengraphLJ2024

	jamBaseData, err := getLJ2024BaseData(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	feedData, err := getLJ2024FeedData(c, 0)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	type JamFeedData struct {
		templates.BaseData
		JamBaseDataLJ2024

		Projects      JamProjectDataLJ2024
		TimelineItems []templates.TimelineItem
	}

	tmpl := JamFeedData{
		BaseData:          baseData,
		JamBaseDataLJ2024: jamBaseData,

		Projects:      feedData.Projects,
		TimelineItems: feedData.TimelineItems,
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2024_lj_feed.html", tmpl, c.Perf)
	return res
}

func JamGuidelines2024_Learning(c *RequestContext) ResponseData {
	baseData := getBaseData(c, hmndata.LJ2024.Name, nil)
	baseData.OpenGraphItems = opengraphLJ2024

	jamBaseData, err := getLJ2024BaseData(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	type JamGuidelinesData struct {
		templates.BaseData
		JamBaseDataLJ2024
	}

	tmpl := JamGuidelinesData{
		BaseData:          baseData,
		JamBaseDataLJ2024: jamBaseData,
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2024_lj_guidelines_index.html", tmpl, c.Perf)
	return res
}

var opengraphLJ2024 = []templates.OpenGraphItem{
	{Property: "og:title", Value: "Learning Jam"},
	{Property: "og:site_name", Value: "Handmade Network"},
	{Property: "og:type", Value: "website"},
	{Property: "og:image", Value: hmnurl.BuildPublic("learningjam2024/2024LJOpenGraph.png", true)},
	{Property: "og:description", Value: "A two-weekend jam where you dive deep into a topic, then share it with the rest of the community."},
	{Property: "og:url", Value: hmnurl.BuildJamIndex2024_Learning()},
	{Name: "twitter:card", Value: "summary_large_image"},
	{Name: "twitter:image", Value: hmnurl.BuildPublic("learningjam2024/2024LJTwitterCard.png", true)},
}

type JamBaseDataLJ2024 struct {
	UserAvatarUrl                string
	DaysUntilStart, DaysUntilEnd int
	JamUrl                       string
	JamFeedUrl                   string
	NewProjectUrl                string
	SubmittedProjectUrl          string
	GuidelinesUrl                string
}

type JamProjectDataLJ2024 struct {
	Projects      []templates.Project
	NewProjectUrl string
}

type JamFeedDataLJ2024 struct {
	Projects      JamProjectDataLJ2024
	TimelineItems []templates.TimelineItem

	projects []hmndata.ProjectAndStuff
}

func getLJ2024BaseData(c *RequestContext) (JamBaseDataLJ2024, error) {
	daysUntilStart := utils.DaysUntil(hmndata.LJ2024.StartTime)
	daysUntilEnd := utils.DaysUntil(hmndata.LJ2024.EndTime)

	tmpl := JamBaseDataLJ2024{
		UserAvatarUrl:  templates.UserAvatarDefaultUrl("dark"),
		DaysUntilStart: daysUntilStart,
		DaysUntilEnd:   daysUntilEnd,
		JamUrl:         hmnurl.BuildJamIndex2024_Learning(),
		JamFeedUrl:     hmnurl.BuildJamFeed2024_Learning(),
		NewProjectUrl:  hmnurl.BuildProjectNewJam(),
		GuidelinesUrl:  hmnurl.BuildJamGuidelines2024_Learning(),
	}

	if c.CurrentUser != nil {
		tmpl.UserAvatarUrl = templates.UserAvatarUrl(c.CurrentUser)
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{hmndata.LJ2024.Slug},
			Limit:    1,
		})
		if err != nil {
			return JamBaseDataLJ2024{}, oops.New(err, "failed to fetch jam projects for current user")
		}
		if len(projects) > 0 {
			urlContext := hmndata.UrlContextForProject(&projects[0].Project)
			tmpl.SubmittedProjectUrl = urlContext.BuildHomepage()
		}
	}

	return tmpl, nil
}

// 0 for no limit on timeline items.
func getLJ2024FeedData(c *RequestContext, maxTimelineItems int) (JamFeedDataLJ2024, error) {
	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.LJ2024.Slug},
	})
	if err != nil {
		return JamFeedDataLJ2024{}, oops.New(err, "failed to fetch jam projects for current user")
	}

	projects := make([]templates.Project, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projects = append(projects, templates.ProjectAndStuffToTemplate(&jp))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	var timelineItems []templates.TimelineItem
	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
			Limit:      maxTimelineItems,
		})
		if err != nil {
			return JamFeedDataLJ2024{}, oops.New(err, "failed to fetch snippets for jam showcase")
		}

		timelineItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, false)
			timelineItems = append(timelineItems, timelineItem)
		}
	}

	return JamFeedDataLJ2024{
		Projects: JamProjectDataLJ2024{
			Projects:      projects,
			NewProjectUrl: hmnurl.BuildProjectNewJam(),
		},
		TimelineItems: timelineItems,

		projects: jamProjects,
	}, nil
}

func getTwitchEmbedUrl(c *RequestContext) string {
	twitchEmbedUrl := ""
	twitchStatus, err := db.QueryOne[models.TwitchLatestStatus](c, c.Conn,
		`
		SELECT $columns
		FROM twitch_latest_status
		WHERE twitch_login = $1
		`,
		"handmadenetwork",
	)
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to query Twitch status for the HMN account")
	} else if twitchStatus.Live {
		hmnUrl, err := url.Parse(config.Config.BaseUrl)
		if err == nil {
			twitchEmbedUrl = fmt.Sprintf("https://player.twitch.tv/?channel=%s&parent=%s", twitchStatus.TwitchLogin, hmnUrl.Hostname())
		}
	}

	return twitchEmbedUrl
}

func JamIndex2023(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilStart := utils.DaysUntil(hmndata.WRJ2023.StartTime)
	daysUntilEnd := utils.DaysUntil(hmndata.WRJ2023.EndTime)

	baseData := getBaseData(c, hmndata.WRJ2023.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2023/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam where we build software from scratch. September 25 - October 1 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2023()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int
		StartTimeUnix, EndTimeUnix   int64

		SubmittedProjectUrl  string
		ProjectSubmissionUrl string
		ShowcaseFeedUrl      string
		ShowcaseJson         string
		TwitchEmbedUrl       string

		JamProjects []templates.Project
	}

	var showcaseItems []templates.TimelineItem
	submittedProjectUrl := ""

	if c.CurrentUser != nil {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{hmndata.WRJ2023.Slug},
			Limit:    1,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
		}
		if len(projects) > 0 {
			urlContext := hmndata.UrlContextForProject(&projects[0].Project)
			submittedProjectUrl = urlContext.BuildHomepage()
		}
	}

	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.WRJ2023.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
			Limit:      12,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}
		showcaseItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, false)
			if timelineItem.CanShowcase {
				showcaseItems = append(showcaseItems, timelineItem)
			}
		}
	}

	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)

	twitchEmbedUrl := ""
	twitchStatus, err := db.QueryOne[models.TwitchLatestStatus](c, c.Conn,
		`
		SELECT $columns
		FROM twitch_latest_status
		WHERE twitch_login = $1
		`,
		"handmadenetwork",
	)
	if err == nil {
		if twitchStatus.Live {
			hmnUrl, err := url.Parse(config.Config.BaseUrl)
			if err == nil {
				twitchEmbedUrl = fmt.Sprintf("https://player.twitch.tv/?channel=%s&parent=%s", twitchStatus.TwitchLogin, hmnUrl.Hostname())
			}
		}
	}

	res.MustWriteTemplate("jam_2023_wrj_index.html", JamPageData{
		BaseData:             baseData,
		DaysUntilStart:       daysUntilStart,
		DaysUntilEnd:         daysUntilEnd,
		StartTimeUnix:        hmndata.WRJ2023.StartTime.Unix(),
		EndTimeUnix:          hmndata.WRJ2023.EndTime.Unix(),
		ProjectSubmissionUrl: hmnurl.BuildProjectNewJam(),
		SubmittedProjectUrl:  submittedProjectUrl,
		ShowcaseFeedUrl:      hmnurl.BuildJamFeed2023(),
		ShowcaseJson:         showcaseJson,
		JamProjects:          pageProjects,
		TwitchEmbedUrl:       twitchEmbedUrl,
	}, c.Perf)
	return res
}

func JamFeed2023(c *RequestContext) ResponseData {
	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.WRJ2023.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	var timelineItems []templates.TimelineItem
	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}

		timelineItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, false)
			timelineItems = append(timelineItems, timelineItem)
		}
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
	}

	type JamFeedData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int

		JamProjects   []templates.Project
		TimelineItems []templates.TimelineItem
	}

	daysUntilStart := utils.DaysUntil(hmndata.WRJ2023.StartTime)
	daysUntilEnd := utils.DaysUntil(hmndata.WRJ2023.EndTime)

	baseData := getBaseData(c, hmndata.WRJ2023.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2023/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. September 25 - October 1 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamFeed2023()},
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2023_wrj_feed.html", JamFeedData{
		BaseData:       baseData,
		DaysUntilStart: daysUntilStart,
		DaysUntilEnd:   daysUntilEnd,
		JamProjects:    pageProjects,
		TimelineItems:  timelineItems,
	}, c.Perf)
	return res
}

func JamIndex2023_Visibility(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilStart := utils.DaysUntil(hmndata.VJ2023.StartTime)
	daysUntilEnd := utils.DaysUntil(hmndata.VJ2023.EndTime)

	baseData := getBaseData(c, hmndata.VJ2023.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2023/opengraph.png", true)},
		{Property: "og:description", Value: "See things in a new way. April 14 - 16."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2023_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2023/TwitterCard.png", true)},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int
		StartTimeUnix, EndTimeUnix   int64

		SubmittedProjectUrl  string
		ProjectSubmissionUrl string
		ShowcaseFeedUrl      string
		ShowcaseJson         string
		RecapUrl             string

		JamProjects []templates.Project
	}

	var showcaseItems []templates.TimelineItem
	submittedProjectUrl := ""

	if c.CurrentUser != nil {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{hmndata.VJ2023.Slug},
			Limit:    1,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
		}
		if len(projects) > 0 {
			urlContext := hmndata.UrlContextForProject(&projects[0].Project)
			submittedProjectUrl = urlContext.BuildHomepage()
		}
	}

	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.VJ2023.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
			Limit:      12,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}
		showcaseItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, false)
			if timelineItem.CanShowcase {
				showcaseItems = append(showcaseItems, timelineItem)
			}
		}
	}

	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)

	res.MustWriteTemplate("jam_2023_vj_index.html", JamPageData{
		BaseData:             baseData,
		DaysUntilStart:       daysUntilStart,
		DaysUntilEnd:         daysUntilEnd,
		StartTimeUnix:        hmndata.VJ2023.StartTime.Unix(),
		EndTimeUnix:          hmndata.VJ2023.EndTime.Unix(),
		ProjectSubmissionUrl: hmnurl.BuildProjectNewJam(),
		SubmittedProjectUrl:  submittedProjectUrl,
		ShowcaseFeedUrl:      hmnurl.BuildJamFeed2023_Visibility(),
		ShowcaseJson:         showcaseJson,
		RecapUrl:             hmnurl.BuildJamRecap2023_Visibility(),
		JamProjects:          pageProjects,
	}, c.Perf)
	return res
}

func JamFeed2023_Visibility(c *RequestContext) ResponseData {
	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.VJ2023.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	var timelineItems []templates.TimelineItem
	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}

		timelineItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, false)
			timelineItems = append(timelineItems, timelineItem)
		}
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
	}

	type JamFeedData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int

		JamUrl        string
		JamProjects   []templates.Project
		TimelineItems []templates.TimelineItem
	}

	daysUntilStart := utils.DaysUntil(hmndata.VJ2023.StartTime)
	daysUntilEnd := utils.DaysUntil(hmndata.VJ2023.EndTime)

	baseData := getBaseData(c, hmndata.VJ2023.Name, nil)

	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2023/opengraph.png", true)},
		{Property: "og:description", Value: "See things in a new way. April 14 - 16."},
		{Property: "og:url", Value: hmnurl.BuildJamFeed2023_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2023/TwitterCard.png", true)},
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2023_vj_feed.html", JamFeedData{
		BaseData:       baseData,
		DaysUntilStart: daysUntilStart,
		DaysUntilEnd:   daysUntilEnd,
		JamUrl:         hmnurl.BuildJamIndex2023_Visibility(),
		JamProjects:    pageProjects,
		TimelineItems:  timelineItems,
	}, c.Perf)
	return res
}

func JamRecap2023_Visibility(c *RequestContext) ResponseData {
	type JamRecapData struct {
		templates.BaseData

		JamUrl   string
		FeedUrl  string
		Ben      templates.User
		PostDate time.Time
	}

	var ben templates.User
	benUser, err := hmndata.FetchUserByUsername(c, c.Conn, c.CurrentUser, "bvisness", hmndata.UsersQuery{})
	if err == nil {
		ben = templates.UserToTemplate(benUser)
	} else if err == db.NotFound {
		ben = templates.UnknownUser
	} else {
		panic("where ben ???")
	}

	baseData := getBaseData(c, hmndata.VJ2023.Name, nil)

	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Visibility Jam"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("visjam2023/opengraph.png", true)},
		// {Property: "og:description", Value: "See things in a new way. April 14 - 16."},
		{Property: "og:url", Value: hmnurl.BuildJamRecap2023_Visibility()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("visjam2023/TwitterCard.png", true)},
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2023_vj_recap.html", JamRecapData{
		BaseData: baseData,
		JamUrl:   hmnurl.BuildJamIndex2023_Visibility(),
		FeedUrl:  hmnurl.BuildJamFeed2023_Visibility(),
		Ben:      ben,
		PostDate: time.Date(2023, 4, 22, 3, 9, 0, 0, time.UTC),
	}, c.Perf)
	return res
}

func JamIndex2022(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilStart := utils.DaysUntil(hmndata.WRJ2022.StartTime)
	daysUntilEnd := utils.DaysUntil(hmndata.WRJ2022.EndTime)

	baseData := getBaseData(c, hmndata.WRJ2022.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2022/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. August 15 - 21 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2022()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int
		StartTimeUnix, EndTimeUnix   int64

		SubmittedProjectUrl  string
		ProjectSubmissionUrl string
		ShowcaseFeedUrl      string
		ShowcaseJson         string

		JamProjects []templates.Project
	}

	var showcaseItems []templates.TimelineItem
	submittedProjectUrl := ""

	if c.CurrentUser != nil {
		projects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{c.CurrentUser.ID},
			JamSlugs: []string{hmndata.WRJ2022.Slug},
			Limit:    1,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
		}
		if len(projects) > 0 {
			urlContext := hmndata.UrlContextForProject(&projects[0].Project)
			submittedProjectUrl = urlContext.BuildHomepage()
		}
	}

	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.WRJ2022.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
			Limit:      12,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}
		showcaseItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, false)
			if timelineItem.CanShowcase {
				showcaseItems = append(showcaseItems, timelineItem)
			}
		}
	}

	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)

	res.MustWriteTemplate("jam_2022_wrj_index.html", JamPageData{
		BaseData:             baseData,
		DaysUntilStart:       daysUntilStart,
		DaysUntilEnd:         daysUntilEnd,
		StartTimeUnix:        hmndata.WRJ2022.StartTime.Unix(),
		EndTimeUnix:          hmndata.WRJ2022.EndTime.Unix(),
		ProjectSubmissionUrl: hmnurl.BuildProjectNewJam(),
		SubmittedProjectUrl:  submittedProjectUrl,
		ShowcaseFeedUrl:      hmnurl.BuildJamFeed2022(),
		ShowcaseJson:         showcaseJson,
		JamProjects:          pageProjects,
	}, c.Perf)
	return res
}

func JamFeed2022(c *RequestContext) ResponseData {
	jamProjects, err := hmndata.FetchProjects(c, c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		JamSlugs: []string{hmndata.WRJ2022.Slug},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam projects for current user"))
	}

	projectIds := make([]int, 0, len(jamProjects))
	for _, jp := range jamProjects {
		projectIds = append(projectIds, jp.Project.ID)
	}

	var timelineItems []templates.TimelineItem
	if len(projectIds) > 0 {
		snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
			ProjectIDs: projectIds,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for jam showcase"))
		}

		timelineItems = make([]templates.TimelineItem, 0, len(snippets))
		for _, s := range snippets {
			timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, false)
			timelineItems = append(timelineItems, timelineItem)
		}
	}

	pageProjects := make([]templates.Project, 0, len(jamProjects))
	for _, p := range jamProjects {
		pageProjects = append(pageProjects, templates.ProjectAndStuffToTemplate(&p))
	}

	type JamFeedData struct {
		templates.BaseData
		DaysUntilStart, DaysUntilEnd int

		JamProjects   []templates.Project
		TimelineItems []templates.TimelineItem
	}

	daysUntilStart := utils.DaysUntil(hmndata.WRJ2022.StartTime)
	daysUntilEnd := utils.DaysUntil(hmndata.WRJ2022.EndTime)

	baseData := getBaseData(c, hmndata.WRJ2022.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2022/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to change the status quo. August 15 - 21 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamFeed2022()},
	}

	var res ResponseData
	res.MustWriteTemplate("jam_2022_wrj_feed.html", JamFeedData{
		BaseData:       baseData,
		DaysUntilStart: daysUntilStart,
		DaysUntilEnd:   daysUntilEnd,
		JamProjects:    pageProjects,
		TimelineItems:  timelineItems,
	}, c.Perf)
	return res
}

func JamIndex2021(c *RequestContext) ResponseData {
	var res ResponseData

	daysUntilJam := utils.DaysUntil(hmndata.WRJ2021.StartTime)
	if daysUntilJam < 0 {
		daysUntilJam = 0
	}

	tagId := -1
	jamTag, err := hmndata.FetchTag(c, c.Conn, hmndata.TagQuery{
		Text: []string{"wheeljam"},
	})
	if err == nil {
		tagId = jamTag.ID
	} else {
		c.Logger.Warn().Err(err).Msg("failed to fetch jam tag; will fetch all snippets as a result")
	}

	snippets, err := hmndata.FetchSnippets(c, c.Conn, c.CurrentUser, hmndata.SnippetQuery{
		Tags: []int{tagId},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch jam snippets"))
	}
	showcaseItems := make([]templates.TimelineItem, 0, len(snippets))
	for _, s := range snippets {
		timelineItem := SnippetToTimelineItem(&s.Snippet, s.Asset, s.DiscordMessage, s.Projects, s.Owner, false)
		if timelineItem.CanShowcase {
			showcaseItems = append(showcaseItems, timelineItem)
		}
	}

	b := c.Perf.StartBlock("SHOWCASE", "Convert to json")
	showcaseJson := templates.TimelineItemsToJSON(showcaseItems)
	b.End()

	baseData := getBaseData(c, hmndata.WRJ2021.Name, nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam2021/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to bring a fresh perspective to old ideas. September 27 - October 3 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex2021()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntil         int
		ShowcaseItemsJSON string
	}

	res.MustWriteTemplate("jam_2021_wrj_index.html", JamPageData{
		BaseData:          baseData,
		DaysUntil:         daysUntilJam,
		ShowcaseItemsJSON: showcaseJson,
	}, c.Perf)
	return res
}

package website

import (
	"html/template"
	"net/http"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func TimeMachine(c *RequestContext) ResponseData {
	baseData := getBaseData(c, "Time Machine", nil)
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Time Machine"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("timemachine/opengraph.png", true)},
		{Property: "og:description", Value: "This summer, dig out your old devices and see what they were actually like to use."},
		{Property: "og:url", Value: hmnurl.BuildTimeMachine()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("timemachine/twittercard.png", true)},
	}

	featured := tmSubmissions[0]
	featured.Title = "Latest Submission"

	type TemplateData struct {
		templates.BaseData
		SubmitUrl          string
		SubmissionsUrl     string
		NumSubmissions     int
		FeaturedSubmission TimeMachineSubmission
	}
	tmpl := TemplateData{
		BaseData:           baseData,
		SubmitUrl:          hmnurl.BuildTimeMachineForm(),
		SubmissionsUrl:     hmnurl.BuildTimeMachineSubmissions(),
		NumSubmissions:     len(tmSubmissions),
		FeaturedSubmission: featured,
	}

	var res ResponseData
	res.MustWriteTemplate("timemachine.html", tmpl, c.Perf)
	return res
}

func TimeMachineSubmissions(c *RequestContext) ResponseData {
	baseData := getBaseData(c, "Time Machine - Submissions", []templates.Breadcrumb{
		{"Time Machine", hmnurl.BuildTimeMachine()},
		{"Submissions", hmnurl.BuildTimeMachineSubmissions()},
	})
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: "Time Machine"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("timemachine/opengraph.png", true)},
		{Property: "og:description", Value: "This summer, dig out your old devices and see what they were actually like to use."},
		{Property: "og:url", Value: hmnurl.BuildTimeMachine()},
		{Name: "twitter:card", Value: "summary_large_image"},
		{Name: "twitter:image", Value: hmnurl.BuildPublic("timemachine/twittercard.png", true)},
	}

	type TemplateData struct {
		templates.BaseData
		MainUrl     string
		SubmitUrl   string
		AtomFeedUrl string
		Submissions []TimeMachineSubmission
	}
	tmpl := TemplateData{
		BaseData:    baseData,
		MainUrl:     hmnurl.BuildTimeMachine(),
		SubmitUrl:   hmnurl.BuildTimeMachineForm(),
		AtomFeedUrl: hmnurl.BuildTimeMachineAtomFeed(),
		Submissions: tmSubmissions,
	}

	var res ResponseData
	res.MustWriteTemplate("timemachine_submissions.html", tmpl, c.Perf)
	return res
}

func TimeMachineForm(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate(
		"timemachine_submit.html",
		getBaseDataAutocrumb(c, "Time Machine"),
		c.Perf,
	)
	return res
}

func TimeMachineFormSubmit(c *RequestContext) ResponseData {
	c.Req.ParseForm()

	mediaUrl := strings.TrimSpace(c.Req.Form.Get("media_url"))
	deviceInfo := strings.TrimSpace(c.Req.Form.Get("device_info"))
	description := strings.TrimSpace(c.Req.Form.Get("description"))

	discordUsername := ""
	if c.CurrentUser.DiscordUser != nil {
		discordUsername = c.CurrentUser.DiscordUser.Username
	}
	err := email.SendTimeMachineEmail(
		hmnurl.BuildUserProfile(c.CurrentUser.Username),
		c.CurrentUser.BestName(),
		c.CurrentUser.Email,
		discordUsername,
		mediaUrl,
		deviceInfo,
		description,
		c.Perf,
	)

	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to send time machine email"))
	}

	return c.Redirect(hmnurl.BuildTimeMachineFormDone(), http.StatusSeeOther)
}

func TimeMachineFormDone(c *RequestContext) ResponseData {
	type TemplateData struct {
		templates.BaseData
		TimeMachineUrl string
	}
	tmpl := TemplateData{
		BaseData:       getBaseDataAutocrumb(c, "Time Machine"),
		TimeMachineUrl: hmnurl.BuildTimeMachine(),
	}

	var res ResponseData
	res.MustWriteTemplate(
		"timemachine_thanks.html",
		tmpl,
		c.Perf,
	)
	return res
}

func TimeMachineAtomFeed(c *RequestContext) ResponseData {
	type TemplateData struct {
		Updated        time.Time
		TimeMachineUrl string
		SubmissionsUrl string
		AtomFeedUrl    string
		LogoUrl        string
		Submissions    []TimeMachineSubmission
	}
	tmpl := TemplateData{
		Updated:        tmSubmissions[0].Date,
		TimeMachineUrl: hmnurl.BuildTimeMachine(),
		SubmissionsUrl: hmnurl.BuildTimeMachineSubmissions(),
		AtomFeedUrl:    hmnurl.BuildTimeMachineAtomFeed(),
		LogoUrl:        hmnurl.BuildPublic("timemachine/twittercard.png", true),
		Submissions:    tmSubmissions,
	}

	var res ResponseData
	res.MustWriteTemplate(
		"timemachine_atom.xml",
		tmpl,
		c.Perf,
	)
	return res
}

type TimeMachineSubmission struct {
	Date        time.Time
	Title       string
	Url         string
	Thumbnail   string
	Permalink   string // generated for feed
	Details     []TimeMachineSubmissionDetail
	Description template.HTML
}

type TimeMachineSubmissionDetail struct {
	Name    string
	Content template.HTML
}

var tmSubmissions = []TimeMachineSubmission{
	{
		Date:      time.Date(2023, 6, 6, 0, 0, 0, 0, time.UTC),
		Title:     "2009 iPod Touch",
		Url:       "https://youtu.be/2eBFk1yV6mE",
		Thumbnail: "timemachine/ipodtouch-dither.gif",
		Details: []TimeMachineSubmissionDetail{
			{"Device", "iPod Touch 3rd gen, model MC008LL"},
			{"Submitted by", "Ben Visness"},
			{"Release year", "2009"},
			{"Processor", "600MHz Samsung S5L8922, single-core"},
			{"Memory", "256MB LPDDR2 @ 200 MHz"},
			{"Operating system", "iOS 5"},
		},
		Description: `
			<p>
				This is the iPod Touch I got when I was 13. It was my first major
				tech purchase and an early device in the iOS lineup. When I
				purchased this I think it was running iOS 3; at this point it has
				iOS 5. I was pleased to see that the battery still holds a charge
				quite well, and it consistently runs at about 30 to 60 frames per
				second.
			</p>
			<p>
				In the video you can see several built-in apps. Media playback
				still works great, and scrubbing around in songs is instantaneous.
				App switching works well. The calculator launches instantly (as
				you would hope). I was shocked to see that the old Google Maps app
				still works - apparently they have kept their old tile-based map
				servers online. It even gave me public transit directions.
			</p>
			<p>
				Overall, I would say this device feels only a hair slower than my
				current iPhone.
			</p>`,
	},
}

func init() {
	for i := range tmSubmissions {
		tmSubmissions[i].Permalink = hmnurl.BuildTimeMachineSubmission(len(tmSubmissions) - i)
	}
}
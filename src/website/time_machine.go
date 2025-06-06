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
	featured.AllSubmissionsUrl = hmnurl.BuildTimeMachineSubmissions()

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
		{"Time Machine", hmnurl.BuildTimeMachine(), nil},
		{"Submissions", hmnurl.BuildTimeMachineSubmissions(), nil},
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
		getBaseData(c, "Time Machine", nil),
		c.Perf,
	)
	return res
}

func TimeMachineFormSubmit(c *RequestContext) ResponseData {
	c.Req.ParseForm()

	mediaUrls := c.Req.Form["media_url"]
	deviceInfo := strings.TrimSpace(c.Req.Form.Get("device_info"))
	description := strings.TrimSpace(c.Req.Form.Get("description"))

	for i := range mediaUrls {
		mediaUrls[i] = strings.TrimSpace(mediaUrls[i])
	}

	discordUsername := ""
	if c.CurrentUser.DiscordUser != nil {
		discordUsername = c.CurrentUser.DiscordUser.Username
	}
	err := email.SendTimeMachineEmail(
		hmnurl.BuildUserProfile(c.CurrentUser.Username),
		c.CurrentUser.BestName(),
		c.CurrentUser.Email,
		discordUsername,
		mediaUrls,
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
		BaseData:       getBaseData(c, "Time Machine", nil),
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
	Thumbnail   TimeMachineThumbnail
	Permalink   string // generated for feed
	Details     []TimeMachineSubmissionDetail
	Description template.HTML
	Horizontal  bool

	AllSubmissionsUrl string
}

type TimeMachineThumbnail struct {
	Filepath      string
	Width, Height float32
}

type TimeMachineSubmissionDetail struct {
	Name    string
	Content template.HTML
}

// You can re-encode videos for the website using some flavor of the following:
//
//     ffmpeg -i input.mp4 -c:v libx264 -profile:v high -preset:v slow -crf:v 24 -c:a aac -b:a 128k -movflags +faststart output.mp4
//
// For 1080p video, you can try 26 or 28 for the video quality and see if it
// looks ok.
//
// Video can be scaled down using `-vf scale=-1:720`, where 720 is the desired
// height of the resulting video. You should probably stick with quality 24 if
// you do this.

var tmSubmissions = []TimeMachineSubmission{
	{
		Date:  time.Date(2023, 7, 21, 0, 0, 0, 0, time.UTC),
		Title: "2010 Lenovo Laptop - \"Leonard\"",
		Url:   "https://www.youtube.com/watch?v=poV9ixJ4KqE",
		Thumbnail: TimeMachineThumbnail{
			Filepath: "timemachine/thumbnails/2023-07-21-thumb.png",
			Width:    298,
			Height:   166,
		},
		Details: []TimeMachineSubmissionDetail{
			{"Device", "Lenovo G570"},
			{"Nickname", "Leonard"},
			{"Submitted by", `<a href="https://handmade.network/m/bvisness" target="_blank">Ben Visness</a>`},
			{"Release year", "2010"},
			{"Processor", "Intel Core i5-2450M @ 2.50GHz"},
			{"Memory", "4GB DDR3"},
			{"Storage", "HDD"},
		},
		Description: `
			<p>
				I have <em>very</em> fond memories of this computer. This was the computer I used for <i>FIRST</i> Robotics back in high school, and the computer we used to drive the robot at the world championships in 2013 and 2014.
			</p>
			<p>
				That doesn't mean it's fast, of course. You can see that it is very sluggish to do many normal tasks, like typing into a Word document.
			</p>
			<p>
				Still, exploring this computer brings back a lot of memories, and funny enough, allows me to show off a lot of different applications. I found old Blender models on here, an old platforming game in Unity that I barely remember, a video called "OBS Test.mp4" on the desktop, and even a definitely legal copy of Counter-Strike 1.6.
			</p>
			<p>
				Naturally, because it's running Windows 7, I was unable to shut it down without running Windows Update. I had to forcibly kill it.
			</p>
		`,
	},
	{
		Date:  time.Date(2023, 7, 19, 0, 0, 0, 0, time.UTC),
		Title: "2012 LG Flip Phone",
		Url:   "https://hmn-assets-2.ams3.cdn.digitaloceanspaces.com/f8e5843a-0be6-4ea5-a980-cc470f2f708e/agus_lg.mp4",
		Thumbnail: TimeMachineThumbnail{
			Filepath: "timemachine/thumbnails/2023-07-19-thumb.png",
			Width:    225,
			Height:   398,
		},
		Details: []TimeMachineSubmissionDetail{
			{"Device", "LG-A133"},
			{"Submitted by", `<a href="https://handmade.network/m/AgusDev" target="_blank">Agustin</a>`},
			{"Release year", "2012"},
			{"Processor", "Unknown"},
			{"Memory", "Unknown"},
		},
		Description: `
			<p>
				One of the first phones I had as a kid, it was a gift from my brother.
			</p>
			<p>
				I always loved how comfortable and light it is to use.
			</p>
			<p>
				<b>Editor's note:</b> This thing runs Java? This thing runs some kind of <i>Assassin's Creed</i> game?? I want to know more about this, but unfortunately I can't find any info about the processor.
			</p>
		`,
		Horizontal: true,
	},
	{
		Date:  time.Date(2023, 7, 4, 0, 0, 0, 0, time.UTC),
		Title: "2011 Philips Media Player",
		Url:   "https://hmn-assets-2.ams3.cdn.digitaloceanspaces.com/a835cf47-9649-4738-bd58-252a6199863b/agus5.mp4",
		Thumbnail: TimeMachineThumbnail{
			Filepath: "timemachine/thumbnails/2023-07-04-thumb.png",
			Width:    226,
			Height:   398,
		},
		Details: []TimeMachineSubmissionDetail{
			{"Device", "Philips SA3047/55"},
			{"Submitted by", `<a href="https://handmade.network/m/AgusDev" target="_blank">Agustin</a>`},
			{"Release year", "2011"},
			{"Processor", "Unknown"},
			{"Memory", "Unknown"},
		},
		Description: `
			<p>
				My mother used to use this a lot when going to study, to record the classes or just listen to music.
			</p>
			<p>
				I remember also using this a bit as a child, when having something for listening to music while doing other stuff was new to me.
			</p>
			<p>
				<b>Editor's note:</b> I couldn't find any info about the processor for this or the device's RAM. If anyone happens to know this info, <a href="mailto:ben@handmade.network">let me know</a>.
			</p>
		`,
		Horizontal: true,
	},
	{
		Date:  time.Date(2023, 6, 16, 0, 0, 0, 0, time.UTC),
		Title: "1992 Intel Professional Workstation",
		Url:   "https://www.youtube.com/watch?v=0uaINJAktLA&t=142s",
		Thumbnail: TimeMachineThumbnail{
			Filepath: "timemachine/thumbnails/2023-06-16-thumb.png",
			Width:    300,
			Height:   169,
		},
		Details: []TimeMachineSubmissionDetail{
			{"Device", "Intel Professional Workstation"},
			{"Submitted by", `<a href="https://www.youtube.com/@NCommander" target="_blank">NCommander</a>`},
			{"Release year", "~1992"},
			{"Processor", "33Mhz 486DX"},
			{"Memory", "Originally 8MiB"},
			{"Architecture", "EISA"},
			{"Operating system", "Shipped with Windows 3.x, OS/2, NetWare, or SCO UNIX"},
		},
		Description: `
			<p>
				This machine was originally used as a workstation at Rutgers University, where it was connected to a NetWare network and used as a fax server at some point. It's hard drive was replaced from one of the stock Maxtor drives to a DEC harddrive that appears to have been pulled out of a VAX.
			</p>
			<p>
				It then made its way to an observatory out on Long Island, and was then used to process radio telemetry data as well as light web surfing throughout the 90s before being retired. It was powered on in 2004 and 2005. I did a multipart series on this machine on YT starting here: <a href="https://www.youtube.com/watch?v=fK7QMAX0BmQ" target="_blank">https://www.youtube.com/watch?v=fK7QMAX0BmQ</a>
			</p>
		`,
	},
	{
		Date:  time.Date(2023, 6, 6, 0, 0, 0, 0, time.UTC),
		Title: "2009 iPod Touch",
		Url:   "https://youtu.be/2eBFk1yV6mE",
		Thumbnail: TimeMachineThumbnail{
			Filepath: "timemachine/thumbnails/2023-06-06-thumb.gif",
			Width:    298,
			Height:   166,
		},
		Details: []TimeMachineSubmissionDetail{
			{"Device", "iPod Touch 3rd gen, model MC008LL"},
			{"Submitted by", `<a href="https://handmade.network/m/bvisness" target="_blank">Ben Visness</a>`},
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

package website

import (
	"net/http"
	"strings"

	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func TimeMachine(c *RequestContext) ResponseData {
	baseData := getBaseDataAutocrumb(c, "Time Machine")
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
		SubmitUrl string
	}
	tmpl := TemplateData{
		BaseData:  baseData,
		SubmitUrl: hmnurl.BuildTimeMachineForm(),
	}

	var res ResponseData
	res.MustWriteTemplate("timemachine.html", tmpl, c.Perf)
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

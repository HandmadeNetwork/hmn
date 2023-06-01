package website

import (
	"net/http"
	"strings"

	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/oops"
)

func TimeMachineForm(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate(
		"time_machine_form.html",
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
	var res ResponseData
	res.MustWriteTemplate(
		"time_machine_form_done.html",
		getBaseDataAutocrumb(c, "Time Machine"),
		c.Perf,
	)
	return res
}

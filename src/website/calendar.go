package website

import (
	"net/http"

	"git.handmade.network/hmn/hmn/src/calendar"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/templates"
)

func CalendarIndex(c *RequestContext) ResponseData {
	type CalData struct {
		templates.BaseData
		Calendars   []string
		Events      []templates.CalendarEvent
		BaseICalUrl string
	}
	events := calendar.GetFutureEvents()

	templateEvents := make([]templates.CalendarEvent, 0, len(events))
	for _, ev := range events {
		templateEvents = append(templateEvents, templates.CalendarEventToTemplate(&ev))
	}

	calNames := []string{}
	for _, c := range config.Config.Calendars {
		calNames = append(calNames, c.Name)
	}

	calendarData := CalData{
		BaseData:    getBaseDataAutocrumb(c, "Calendar"),
		Calendars:   calNames,
		Events:      templateEvents,
		BaseICalUrl: hmnurl.BuildCalendarICal(),
	}
	var res ResponseData
	res.MustWriteTemplate("calendar_index.html", calendarData, c.Perf)
	return res
}

func CalendarICal(c *RequestContext) ResponseData {
	query := c.Req.URL.Query()
	cals := make([]string, 0, len(query))
	for key := range query {
		cals = append(cals, key)
	}
	calBytes, err := calendar.GetICal(cals)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}
	var res ResponseData
	res.Write(calBytes)
	return res
}

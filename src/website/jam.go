package website

import (
	"time"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/templates"
)

func JamIndex(c *RequestContext) ResponseData {
	var res ResponseData

	jamStartTime := time.Date(2021, 9, 27, 0, 0, 0, 0, time.UTC)
	daysUntil := jamStartTime.YearDay() - time.Now().UTC().YearDay()
	if daysUntil < 0 {
		daysUntil = 0
	}

	baseData := getBaseDataAutocrumb(c, "Wheel Reinvention Jam")
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade.Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to bring a fresh perspective to old ideas. September 27 - October 3 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex()},
	}

	type JamPageData struct {
		templates.BaseData
		DaysUntil int
	}

	res.MustWriteTemplate("wheeljam_index.html", JamPageData{
		BaseData:  baseData,
		DaysUntil: daysUntil,
	}, c.Perf)
	return res
}

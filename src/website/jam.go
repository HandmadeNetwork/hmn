package website

import (
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/templates"
)

func JamIndex(c *RequestContext) ResponseData {
	var res ResponseData

	/*
		ogimagepath = '%swheeljam/opengraph.png' % (settings.STATIC_URL)
		ogimageurl = urljoin(current_site_host(), ogimagepath)
	*/

	baseData := getBaseData(c)
	baseData.Title = "Wheel Reinvention Jam"
	baseData.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:site_name", Value: "Handmade.Network"},
		{Property: "og:type", Value: "website"},
		{Property: "og:image", Value: hmnurl.BuildPublic("wheeljam/opengraph.png", true)},
		{Property: "og:description", Value: "A one-week jam to bring a fresh perspective to old ideas. September 27 - October 3 on Handmade Network."},
		{Property: "og:url", Value: hmnurl.BuildJamIndex()},
	}

	res.MustWriteTemplate("wheeljam_index.html", baseData, c.Perf)
	return res
}

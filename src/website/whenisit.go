package website

import (
	"fmt"
	"strconv"

	"git.handmade.network/hmn/hmn/src/templates"
)

type WhenIsItData struct {
	templates.BaseData
	Timestamp int
	Name      string
	Url       string
}

func WhenIsIt(c *RequestContext) ResponseData {
	timestampStr := c.Req.URL.Query().Get("t")
	timestamp := 0
	hasTimestamp := false

	if timestampStr != "" {
		var err error
		timestamp, err = strconv.Atoi(timestampStr)
		hasTimestamp = (err == nil)
	}

	baseData := getBaseDataAutocrumb(c, "When is it?")

	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:title",
		Value:    baseData.Title,
	})
	baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
		Property: "og:url",
		Value:    c.FullUrl(),
	})

	if hasTimestamp {
		name := c.Req.URL.Query().Get("n")
		url := c.Req.URL.Query().Get("u")

		if name != "" {
			baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
				Property: "og:description",
				Value:    fmt.Sprintf("Find out when %s starts.", name),
			})
		}

		var res ResponseData
		res.MustWriteTemplate("whenisit.html", WhenIsItData{
			BaseData:  baseData,
			Timestamp: timestamp,
			Name:      name,
			Url:       url,
		}, c.Perf)
		return res
	} else {
		baseData.OpenGraphItems = append(baseData.OpenGraphItems, templates.OpenGraphItem{
			Property: "og:description",
			Value:    "A countdown timer",
		})
		var res ResponseData
		res.MustWriteTemplate("whenisit_setup.html", baseData, c.Perf)
		return res
	}
}

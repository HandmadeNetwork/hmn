package website

import (
	"fmt"
	"strings"

	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/templates"
)

type ExpoTemplateData struct {
	templates.BaseData
	Expo         hmndata.Expo
	BuyTicketUrl string
}

func ExpoIndex(c *RequestContext) ResponseData {
	slug := c.PathParams["urlslug"]

	expo, ok := findExpoBySlug(slug)
	if !ok {
		return FourOhFour(c)
	}

	templateData := ExpoTemplateData{
		BaseData:     getBaseData(c, expo.Name, nil),
		Expo:         expo,
		BuyTicketUrl: hmnurl.BuildTicketsPurchase(expo.UrlSlug),
	}
	templateName := fmt.Sprintf("expo_%s_index.html", expo.TemplateName)

	var res ResponseData
	res.MustWriteTemplate(templateName, templateData, c.Perf)
	return res
}

func ExpoTicketPurchaseSuccess(c *RequestContext) ResponseData {
	urlSlug := c.PathParams["urlslug"]
	event, found := findTicketEventBySlug(urlSlug)
	if !found {
		return FourOhFour(c)
	}

	// TODO: Render a real nice success page that tells them to check their email or whatever.
	_ = event

	return FourOhFour(c)
}

type ExpoAdminTemplateData struct {
	templates.BaseData
}

func findExpoBySlug(urlSlug string) (hmndata.Expo, bool) {
	urlSlug = strings.ToLower(urlSlug)
	for _, e := range hmndata.AllExpos {
		if strings.ToLower(e.UrlSlug) == urlSlug {
			return e, true
		}
	}

	return hmndata.Expo{}, false
}

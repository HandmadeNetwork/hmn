package website

import (
	"git.handmade.network/hmn/hmn/src/templates"
)

func HSFLanding(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_landing.html", getHSFBaseData(c, "", nil), c.Perf)
	return res
}

func HSFDetails(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_details.html", getHSFBaseData(c, "Details", nil), c.Perf)
	return res
}

func HSFMembership(c *RequestContext) ResponseData {
	baseData := getHSFBaseData(c, "Membership", nil)
	baseData.HideMembershipCTA = true

	var res ResponseData
	res.MustWriteTemplate("hsf_membership.html", baseData, c.Perf)
	return res
}

func getHSFBaseData(c *RequestContext, title string, breadcrumbs []templates.Breadcrumb) templates.BaseData {
	baseData := getBaseData(c, title, breadcrumbs)
	baseData.SiteTitleOverride = "Handmade Software Foundation"
	baseData.ShowFoundationFooter = true

	return baseData
}

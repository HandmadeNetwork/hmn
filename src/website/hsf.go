package website

import (
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/stripe/stripe-go/v84"
)

func HSFLanding(c *RequestContext) ResponseData {
	var res ResponseData
	res.MustWriteTemplate("hsf_landing.html", getHSFBaseData(c, "", nil), c.Perf)
	return res
}

func HSFDetails(c *RequestContext) ResponseData {
	breadcrumbs := []templates.Breadcrumb{
		hsfBaseBreadcrumb,
		{Name: "Details", Url: hmnurl.BuildHSFDetails()},
	}

	var res ResponseData
	res.MustWriteTemplate("hsf_details.html", getHSFBaseData(c, "Details", breadcrumbs), c.Perf)
	return res
}

func HSFMembership(c *RequestContext) ResponseData {
	if c.Req.URL.Query().Get("payment_method_updated") == "1" && c.CurrentUser != nil {
		sc := stripe.NewClient(config.Config.Stripe.SecretKey)
		if err := retryPastDueSubscriptionPayment(c, c.Conn, sc, c.CurrentUser); err != nil {
			c.Logger.Warn().Err(err).Msg("failed to retry subscription payment after billing portal return")
		}
		if user, err := db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE id = $1", c.CurrentUser.ID); err == nil {
			c.CurrentUser = user
		}
	}

	breadcrumbs := []templates.Breadcrumb{
		hsfBaseBreadcrumb,
		{Name: "Membership", Url: hmnurl.BuildHSFMembership()},
	}

	baseData := getHSFBaseData(c, "Membership", breadcrumbs)
	baseData.HideMembershipCTA = true

	var res ResponseData
	res.MustWriteTemplate("hsf_membership.html", buildMembershipPageData(c, baseData), c.Perf)
	return res
}

var hsfBaseBreadcrumb = templates.Breadcrumb{Name: "Handmade Software Foundation", Url: hmnurl.BuildHSFLanding()}

func getHSFBaseData(c *RequestContext, title string, breadcrumbs []templates.Breadcrumb) templates.BaseData {
	baseData := getBaseData(c, title, breadcrumbs)
	baseData.SiteTitleOverride = "Handmade Software Foundation"
	baseData.ShowFoundationFooter = true

	return baseData
}

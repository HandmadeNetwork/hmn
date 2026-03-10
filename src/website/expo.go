package website

import (
	"fmt"
	"strings"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/checkout/session"
)

func ExpoIndex(c *RequestContext) ResponseData {
	slug := c.PathParams["urlslug"]

	expo, ok := findExpoBySlug(slug)
	if !ok {
		return FourOhFour(c)
	}
	event, ok := hmndata.FindTicketEventBySlug(slug)
	utils.Assert(ok)

	metadata, err := fetchTicketMetadataForEvent(c, c.Conn, &event)
	if err != nil {
		// We continue on from this and just render possibly brokenly
		c.Logger.Error().Err(err).Msg("Failed to look up event metadata on expo page")
	}

	type Tmpl struct {
		templates.BaseData
		Expo                hmndata.Expo
		BuyTicketUrl        string
		ContinuePurchaseUrl string
		ViewTicketUrl       string
		CanPurchase         bool
		SoldOut             bool
	}

	canPurchase := metadata.MaxTickets > 0 && !metadata.Presale
	soldOut := canPurchase && metadata.RemainingTicketsForSale() <= 0
	tmpl := Tmpl{
		BaseData:     getBaseData(c, expo.Name, nil),
		Expo:         expo,
		BuyTicketUrl: hmnurl.BuildTicketPurchase(expo.UrlSlug),
		CanPurchase:  canPurchase,
		SoldOut:      soldOut,
	}
	tmpl.OpenGraphItems = []templates.OpenGraphItem{
		{Property: "og:title", Value: tmpl.Title},
		{Property: "og:url", Value: hmnurl.BuildExpo(expo.UrlSlug, "")},
		{Property: "og:type", Value: "website"},
		{Property: "og:site_name", Value: "Handmade Network"},
		{Property: "og:description", Value: expo.Description},
	}
	if expo.Image != "" {
		tmpl.OpenGraphItems = append(tmpl.OpenGraphItems, []templates.OpenGraphItem{
			{Property: "og:image", Value: expo.Image},
			{Name: "twitter:card", Value: "summary_large_image"},
			{Name: "twitter:image", Value: expo.Image},
		}...)
	}

	if c.CurrentUser != nil {
		userTicket, err := fetchTicketForUser(c, c.Conn, &event, c.CurrentUser.ID)
		if err == db.NotFound {
			// No ticket, no problem
		} else if err != nil {
			c.Logger.Error().Err(err).Msg("Failed to look up user ticket on expo home page")
		} else {
			if userTicket.Pending {
				// Not ideal, but we must fetch the checkout session from Stripe to get its URL. Since this
				// only happens for users who have started (but not finished) the checkout flow, this
				// shouldn't be a particularly big deal.
				if sess, err := session.Get(userTicket.StripeCheckoutSessionID, &stripe.CheckoutSessionParams{}); err == nil {
					tmpl.ContinuePurchaseUrl = sess.URL
				} else {
					c.Logger.Warn().Err(err).Msg("Failed to get user checkout session to continue purchase flow")
				}
			} else {
				// The user has already purchased a ticket! Hooray!
				tmpl.ViewTicketUrl = hmnurl.BuildTicketSingle(userTicket.ID.String())
			}
		}
	}

	tmpl.ForceDark = true
	templateName := fmt.Sprintf("expo_%s_index.html", expo.TemplateName)

	var res ResponseData
	res.MustWriteTemplate(templateName, tmpl, c.Perf)
	return res
}

func ExpoTicketPurchaseSuccess(c *RequestContext) ResponseData {
	slug := c.PathParams["urlslug"]
	expo, ok := findExpoBySlug(slug)
	if !ok {
		return FourOhFour(c)
	}
	event, ok := hmndata.FindTicketEventBySlug(slug)
	utils.Assert(ok)

	type Tmpl struct {
		templates.BaseData
		ExpoURL string
	}
	tmpl := Tmpl{
		BaseData: getBaseData(c, event.Name, nil),
		ExpoURL:  hmnurl.BuildExpo(expo.UrlSlug, ""),
	}

	tmpl.ForceDark = true
	templateName := fmt.Sprintf("expo_%s_success.html", expo.TemplateName)

	var res ResponseData
	res.MustWriteTemplate(templateName, tmpl, c.Perf)
	return res
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

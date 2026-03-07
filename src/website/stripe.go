package website

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

func init() {
	// Use the global client
	stripe.Key = config.Config.Stripe.SecretKey
}

func StripeWebhook(c *RequestContext) ResponseData {
	const MaxBodyBytes = 65536
	payload, err := io.ReadAll(io.LimitReader(c.Req.Body, MaxBodyBytes))
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "oversize Stripe payload, probably"))
	}

	event, err := webhook.ConstructEventWithOptions(
		payload,
		c.Req.Header.Get("Stripe-Signature"), config.Config.Stripe.WebhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "failed to verify Stripe webhook signature"))
	}

	c.Logger.Info().Str("type", string(event.Type)).Msg("received Stripe webhook")

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return c.ErrorResponse(http.StatusBadRequest, oops.New(err, "bad JSON in stripe webhook"))
		}
		return stripeCheckoutSessionCompleted(c, &session)
	default:
		return ResponseData{StatusCode: http.StatusOK}
	}
}

func stripeCheckoutSessionCompleted(c *RequestContext, session *stripe.CheckoutSession) ResponseData {
	for _, li := range session.LineItems.Data {
		// NOTE(ben): We figure out which event the purchase corresponds to here, rather than in
		// tickets.go, because eventually we will also have HSF subscriptions in here and we will need
		// to disambiguate up front.
		var event *hmndata.Event
		for _, e := range hmndata.AllTicketEvents {
			if e.StripeProductID == li.Price.Product.ID {
				event = &e
				break
			}
		}
		if event != nil {
			return c.ErrorResponse(http.StatusInternalServerError, fmt.Errorf("couldn't find ticket event for Stripe product %s", li.Price.Product.ID))
		}
		if err := ticketsEventBuyPurchased_Stripe(session, *event); err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to process ticket purchase"))
		}
	}

	return ResponseData{StatusCode: http.StatusOK}
}

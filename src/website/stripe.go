package website

import (
	"encoding/json"
	"io"
	"net/http"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
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
		return c.JSONErrorResponse(http.StatusBadRequest, oops.New(err, "oversize Stripe payload, probably"))
	}

	event, err := webhook.ConstructEventWithOptions(
		payload,
		c.Req.Header.Get("Stripe-Signature"), config.Config.Stripe.WebhookSecret,
		webhook.ConstructEventOptions{
			IgnoreAPIVersionMismatch: true,
		},
	)
	if err != nil {
		return c.JSONErrorResponse(http.StatusBadRequest, oops.New(err, "failed to verify Stripe webhook signature"))
	}

	c.Logger.Info().Str("type", string(event.Type)).Msg("received Stripe webhook")

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return c.JSONErrorResponse(http.StatusBadRequest, oops.New(err, "bad JSON in stripe webhook"))
		}
		return stripeCheckoutSessionCompleted(c, &session)
	case "checkout.session.expired":
		var session stripe.CheckoutSession
		err := json.Unmarshal(event.Data.Raw, &session)
		if err != nil {
			return c.JSONErrorResponse(http.StatusBadRequest, oops.New(err, "bad JSON in stripe webhook"))
		}
		return stripeCheckoutSessionExpired(c, &session)
	default:
		return ResponseData{StatusCode: http.StatusOK}
	}
}

func stripeCheckoutSessionCompleted(c *RequestContext, session *stripe.CheckoutSession) ResponseData {
	// Different Stripe checkout flows may dispatch to different things.

	ticket, err := hmndata.FetchTicket(c, c.Conn, hmndata.TicketQuery{
		StripeCheckoutSessionID: session.ID,
	})
	if err == nil {
		err := confirmStripeTicketPurchase(c, c.Conn, session, ticket)
		if err != nil {
			return c.JSONErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to process ticket purchase"))
		}
		return c.JSONResponse(http.StatusOK, map[string]any{
			"confirmedTicket": ticket.ID,
		})
	} else if err == db.NotFound {
		// all good, move on to other checkout things
	} else {
		return c.JSONErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up ticket for checkout session"))
	}

	c.Logger.Warn().
		Str("session ID", session.ID).
		Str("payment intent ID", session.PaymentIntent.ID).
		Msg("Unknown checkout session! What could it mean???")
	return ResponseData{StatusCode: http.StatusOK}
}

func stripeCheckoutSessionExpired(c *RequestContext, session *stripe.CheckoutSession) ResponseData {
	// Different Stripe checkout flows may dispatch to different things.

	numDeleted, err := cancelPendingTicketsForCheckoutSession(c, c.Conn, session)
	if err != nil {
		return c.JSONErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to clear tickets for expired checkout session"))
	}

	return c.JSONResponse(http.StatusOK, map[string]any{
		"ticketsDeleted": numDeleted,
	})
}

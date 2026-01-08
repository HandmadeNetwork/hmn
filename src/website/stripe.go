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

	sc := stripe.NewClient(config.Config.Stripe.SecretKey)

	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err == nil {
			if session.Mode == stripe.CheckoutSessionModeSubscription {
				handleCheckoutSessionCompleted(c, sc, &session)
			} else {
				stripeCheckoutSessionCompleted(c, &session)
			}
		} else {
			c.Logger.Error().Err(err).Msg("failed to unmarshal checkout.session.completed")
		}
	case "checkout.session.expired":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err == nil {
			stripeCheckoutSessionExpired(c, &session)
		} else {
			c.Logger.Error().Err(err).Msg("failed to unmarshal checkout.session.expired")
		}
	case "customer.subscription.created":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err == nil {
			handleSubscriptionCreated(c, sc, &sub)
		} else {
			c.Logger.Error().Err(err).Msg("failed to unmarshal customer.subscription.created")
		}
	case "customer.subscription.updated":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err == nil {
			c.Logger.Trace().RawJSON("sub_json", event.Data.Raw).Msg("received subscription update JSON")
			handleSubscriptionUpdated(c, sc, &sub)
		} else {
			c.Logger.Error().Err(err).Msg("failed to unmarshal customer.subscription.updated")
		}
	case "customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err == nil {
			handleSubscriptionDeleted(c, sc, &sub)
		} else {
			c.Logger.Error().Err(err).Msg("failed to unmarshal customer.subscription.deleted")
		}
	case "invoice.paid":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err == nil {
			handleInvoicePaid(c, sc, &inv)
		} else {
			c.Logger.Error().Err(err).Msg("failed to unmarshal invoice.paid")
		}
	case "invoice.payment_failed":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err == nil {
			handleInvoicePaymentFailed(c, sc, &inv)
		} else {
			c.Logger.Error().Err(err).Msg("failed to unmarshal invoice.payment_failed")
		}
	}

	return ResponseData{StatusCode: http.StatusOK}
}

func stripeCheckoutSessionCompleted(c *RequestContext, session *stripe.CheckoutSession) {
	// Different Stripe checkout flows may dispatch to different things.

	ticket, err := hmndata.FetchTicket(c, c.Conn, hmndata.TicketQuery{
		StripeCheckoutSessionID: session.ID,
	})
	if err == nil {
		err := confirmStripeTicketPurchase(c, c.Conn, session, ticket)
		if err != nil {
			c.Logger.Error().Err(err).Msg("failed to process ticket purchase")
		}
		return
	} else if err == db.NotFound {
		// all good, move on to other checkout things
	} else {
		c.Logger.Error().Err(err).Msg("failed to look up ticket for checkout session")
		return
	}

	c.Logger.Warn().
		Str("session ID", session.ID).
		Str("payment intent ID", session.PaymentIntent.ID).
		Msg("Unknown checkout session! What could it mean???")
}

func stripeCheckoutSessionExpired(c *RequestContext, session *stripe.CheckoutSession) {
	// Different Stripe checkout flows may dispatch to different things.

	_, err := cancelPendingTicketsForCheckoutSession(c, c.Conn, session)
	if err != nil {
		c.Logger.Error().Err(err).Msg("failed to clear tickets for expired checkout session")
	}
}

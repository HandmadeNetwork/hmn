package website

import (
	"encoding/json"
	"net/http"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/stripe/stripe-go/v84"
)

// handleTicketStripeEvent handles ticket Stripe webhook events.
func handleTicketStripeEvent(c *RequestContext, sc *stripe.Client, event *stripe.Event) ResponseData {
	switch event.Type {
	case "checkout.session.completed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return c.JSONErrorResponse(http.StatusBadRequest, oops.New(err, "bad JSON in stripe webhook"))
		}
		return ticketCheckoutSessionCompleted(c, &session)
	case "checkout.session.expired":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return c.JSONErrorResponse(http.StatusBadRequest, oops.New(err, "bad JSON in stripe webhook"))
		}
		return ticketCheckoutSessionExpired(c, &session)
	}
	return ResponseData{StatusCode: http.StatusOK}
}

func ticketCheckoutSessionCompleted(c *RequestContext, session *stripe.CheckoutSession) ResponseData {
	ticket, err := hmndata.FetchTicket(c, c.Conn, hmndata.TicketQuery{
		StripeCheckoutSessionID: session.ID,
	})
	if err == db.NotFound {
		c.Logger.Warn().
			Str("session ID", session.ID).
			Msg("checkout session matched a ticket product but no ticket row exists")
		return ResponseData{StatusCode: http.StatusOK}
	} else if err != nil {
		return c.JSONErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up ticket for checkout session"))
	}

	if err := confirmStripeTicketPurchase(c, c.Conn, session, ticket); err != nil {
		return c.JSONErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to process ticket purchase"))
	}
	return c.JSONResponse(http.StatusOK, map[string]any{
		"confirmedTicket": ticket.ID,
	})
}

func ticketCheckoutSessionExpired(c *RequestContext, session *stripe.CheckoutSession) ResponseData {
	numDeleted, err := cancelPendingTicketsForCheckoutSession(c, c.Conn, session)
	if err != nil {
		return c.JSONErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to clear tickets for expired checkout session"))
	}
	return c.JSONResponse(http.StatusOK, map[string]any{
		"ticketsDeleted": numDeleted,
	})
}

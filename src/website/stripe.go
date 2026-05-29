package website

import (
	"context"
	"encoding/json"
	"io"
	"net/http"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/stripe/stripe-go/v84"
	"github.com/stripe/stripe-go/v84/webhook"
)

func init() {
	stripe.Key = config.Config.Stripe.SecretKey
}

// StripeWebhook verifies and routes all Stripe webhook events.
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

	if isMembershipGracePaymentRetryEvent(&event) {
		handleMembershipGracePaymentRetryWebhook(c, sc, &event)
	}

	if tryHandleMembershipPaymentIntentWebhook(c, sc, &event) {
		return ResponseData{StatusCode: http.StatusOK}
	}

	priceIDs, err := stripePriceIDsForEvent(c, sc, &event)
	if err != nil {
		c.Logger.Error().Err(err).Str("type", string(event.Type)).Msg("failed to resolve price IDs for stripe event")
		return ResponseData{StatusCode: http.StatusOK}
	}

	kind, err := classifyStripePriceIDs(c, c.Conn, priceIDs)
	if err != nil {
		c.Logger.Error().Err(err).Msg("failed to classify stripe webhook by price")
		return ResponseData{StatusCode: http.StatusOK}
	}

	switch kind {
	case stripeWebhookKindTicket:
		return handleTicketStripeEvent(c, sc, &event)
	case stripeWebhookKindMembership:
		handleMembershipStripeEvent(c, sc, &event)
		return ResponseData{StatusCode: http.StatusOK}
	default:
		c.Logger.Warn().
			Str("type", string(event.Type)).
			Strs("prices", priceIDs).
			Msg("Stripe webhook did not match any known ticket or membership price; ignoring")
		return ResponseData{StatusCode: http.StatusOK}
	}
}

type stripeWebhookKind int

const (
	stripeWebhookKindUnknown stripeWebhookKind = iota
	stripeWebhookKindTicket
	stripeWebhookKindMembership
)

func membershipPriceIDAllowed(priceID string) bool {
	if priceID == "" {
		return false
	}
	for _, id := range membershipWebhookPriceIDs() {
		if priceID == id {
			return true
		}
	}
	return false
}

func membershipWebhookPriceIDs() []string {
	var out []string
	seen := map[string]struct{}{}
	add := func(id string) {
		if id == "" {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	add(config.Config.Stripe.PriceID)
	for _, id := range config.Config.Stripe.MembershipAlternatePriceIDs {
		add(id)
	}
	return out
}

func classifyStripePriceIDs(ctx context.Context, conn db.ConnOrTx, priceIDs []string) (stripeWebhookKind, error) {
	if len(priceIDs) == 0 {
		return stripeWebhookKindUnknown, nil
	}

	membership := map[string]struct{}{}
	for _, id := range membershipWebhookPriceIDs() {
		membership[id] = struct{}{}
	}
	for _, id := range priceIDs {
		if _, ok := membership[id]; ok {
			return stripeWebhookKindMembership, nil
		}
	}

	ticketPriceIDs, err := db.QueryScalar[string](ctx, conn, `
		SELECT stripe_price_id
		FROM ticket_metadata
		WHERE stripe_price_id <> ''
	`)
	if err != nil {
		return stripeWebhookKindUnknown, oops.New(err, "failed to load ticket price ids")
	}

	known := make(map[string]struct{}, len(ticketPriceIDs))
	for _, id := range ticketPriceIDs {
		known[id] = struct{}{}
	}
	for _, id := range priceIDs {
		if _, ok := known[id]; ok {
			return stripeWebhookKindTicket, nil
		}
	}

	return stripeWebhookKindUnknown, nil
}

func stripePriceIDsForEvent(ctx context.Context, sc *stripe.Client, event *stripe.Event) ([]string, error) {
	switch event.Type {
	case "checkout.session.completed", "checkout.session.expired", "checkout.session.async_payment_failed", "checkout.session.async_payment_succeeded":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return nil, oops.New(err, "bad checkout session JSON in stripe webhook")
		}
		return checkoutSessionPriceIDs(ctx, sc, &session)

	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted", "customer.subscription.paused", "customer.subscription.resumed", "customer.subscription.trial_will_end":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return nil, oops.New(err, "bad subscription JSON in stripe webhook")
		}
		return subscriptionPriceIDs(&sub), nil

	case "invoice.paid", "invoice.payment_failed", "invoice.payment_succeeded", "invoice.finalized", "invoice.upcoming":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return nil, oops.New(err, "bad invoice JSON in stripe webhook")
		}
		return invoicePriceIDs(&inv), nil
	}

	return nil, nil
}

func checkoutSessionPriceIDs(ctx context.Context, sc *stripe.Client, session *stripe.CheckoutSession) ([]string, error) {
	var ids []string
	seen := map[string]struct{}{}

	iter := sc.V1CheckoutSessions.ListLineItems(ctx, &stripe.CheckoutSessionListLineItemsParams{
		Session: stripe.String(session.ID),
	})
	var iterErr error
	iter(func(item *stripe.LineItem, err error) bool {
		if err != nil {
			iterErr = err
			return false
		}
		if item == nil || item.Price == nil || item.Price.ID == "" {
			return true
		}
		pid := item.Price.ID
		if _, ok := seen[pid]; ok {
			return true
		}
		seen[pid] = struct{}{}
		ids = append(ids, pid)
		return true
	})
	if iterErr != nil {
		return nil, oops.New(iterErr, "failed to list checkout session line items")
	}
	return ids, nil
}

func subscriptionPriceIDs(sub *stripe.Subscription) []string {
	if sub == nil || sub.Items == nil {
		return nil
	}
	var ids []string
	seen := map[string]struct{}{}
	for _, item := range sub.Items.Data {
		if item == nil || item.Price == nil || item.Price.ID == "" {
			continue
		}
		pid := item.Price.ID
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		ids = append(ids, pid)
	}
	return ids
}

func invoicePriceIDs(inv *stripe.Invoice) []string {
	if inv == nil || inv.Lines == nil {
		return nil
	}
	var ids []string
	seen := map[string]struct{}{}
	for _, line := range inv.Lines.Data {
		if line == nil || line.Pricing == nil || line.Pricing.PriceDetails == nil {
			continue
		}
		pid := line.Pricing.PriceDetails.Price
		if pid == "" {
			continue
		}
		if _, ok := seen[pid]; ok {
			continue
		}
		seen[pid] = struct{}{}
		ids = append(ids, pid)
	}
	return ids
}

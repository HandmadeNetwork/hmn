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

func beginStripeWebhookEvent(ctx context.Context, conn db.ConnOrTx, event *stripe.Event) (bool, error) {
	tag, err := conn.Exec(ctx, `
		INSERT INTO stripe_webhook_event (event_id, event_type, status, last_error, updated_at, processed_at)
		VALUES ($1, $2, 'processing', NULL, NOW(), NULL)
		ON CONFLICT (event_id) DO NOTHING
	`, event.ID, string(event.Type))
	if err != nil {
		return false, oops.New(err, "failed to insert stripe webhook event id")
	}
	if tag.RowsAffected() == 1 {
		// First claimant for this event ID.
		return true, nil
	}

	status, err := db.QueryOneScalar[string](ctx, conn, `
		SELECT status
		FROM stripe_webhook_event
		WHERE event_id = $1
	`, event.ID)
	if err != nil {
		return false, oops.New(err, "failed to read Stripe webhook event state")
	}
	switch status {
	case "processed", "processing":
		return false, nil
	case "failed":
		tag, err = conn.Exec(ctx, `
		UPDATE stripe_webhook_event
		SET
			event_type = $2,
			status = 'processing',
			last_error = NULL,
			updated_at = NOW(),
			processed_at = NULL
		WHERE event_id = $1
		  AND status = 'failed'
	`, event.ID, string(event.Type))
		if err != nil {
			return false, oops.New(err, "failed to mark stripe webhook event as processing")
		}
		return tag.RowsAffected() == 1, nil
	default:
		return false, nil
	}
}

func shouldProcessMembershipEventOrder(ctx context.Context, conn db.ConnOrTx, event *stripe.Event) (bool, error) {
	customerID := membershipEventCustomerID(event)
	if customerID == "" {
		return true, nil
	}

	createdAt := event.Created
	if createdAt <= 0 {
		return true, nil
	}

	tag, err := conn.Exec(ctx, `
		INSERT INTO stripe_membership_event_cursor (customer_id, last_event_created, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (customer_id) DO UPDATE
		SET
			last_event_created = EXCLUDED.last_event_created,
			updated_at = NOW()
		WHERE stripe_membership_event_cursor.last_event_created <= EXCLUDED.last_event_created
	`, customerID, createdAt)
	if err != nil {
		return false, oops.New(err, "failed to update membership event cursor")
	}
	return tag.RowsAffected() == 1, nil
}

func membershipEventCustomerID(event *stripe.Event) string {
	if event == nil {
		return ""
	}
	switch event.Type {
	case stripe.EventTypePaymentIntentProcessing,
		stripe.EventTypePaymentIntentRequiresAction,
		stripe.EventTypePaymentIntentPaymentFailed:
		var pi stripe.PaymentIntent
		if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
			return ""
		}
		if pi.Customer == nil {
			return ""
		}
		return pi.Customer.ID
	case "customer.subscription.created",
		"customer.subscription.updated",
		"customer.subscription.deleted":
		var sub stripe.Subscription
		if err := json.Unmarshal(event.Data.Raw, &sub); err != nil {
			return ""
		}
		if sub.Customer == nil {
			return ""
		}
		return sub.Customer.ID
	case "invoice.paid", "invoice.payment_failed":
		var inv stripe.Invoice
		if err := json.Unmarshal(event.Data.Raw, &inv); err != nil {
			return ""
		}
		if inv.Customer == nil {
			return ""
		}
		return inv.Customer.ID
	case "checkout.session.completed", "checkout.session.async_payment_succeeded", "checkout.session.async_payment_failed":
		var session stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &session); err != nil {
			return ""
		}
		if session.Customer == nil {
			return ""
		}
		return session.Customer.ID
	case "payment_method.attached":
		var pm stripe.PaymentMethod
		if err := json.Unmarshal(event.Data.Raw, &pm); err != nil {
			return ""
		}
		if pm.Customer == nil {
			return ""
		}
		return pm.Customer.ID
	case "customer.updated":
		var customer stripe.Customer
		if err := json.Unmarshal(event.Data.Raw, &customer); err != nil {
			return ""
		}
		return customer.ID
	default:
		return ""
	}
}

func isMembershipPaymentIntentEvent(event *stripe.Event) bool {
	if event == nil {
		return false
	}
	switch event.Type {
	case stripe.EventTypePaymentIntentProcessing,
		stripe.EventTypePaymentIntentRequiresAction,
		stripe.EventTypePaymentIntentPaymentFailed:
		return true
	default:
		return false
	}
}

func checkMembershipEventOrder(c *RequestContext, event *stripe.Event) bool {
	shouldProcess, err := shouldProcessMembershipEventOrder(c, c.Conn, event)
	if err != nil {
		c.Logger.Error().Err(err).Str("eventID", event.ID).Msg("failed membership event ordering guard")
		return false
	}
	if !shouldProcess {
		c.Logger.Info().Str("eventID", event.ID).Str("type", string(event.Type)).Msg("stale membership event by created timestamp; ignoring")
		return false
	}
	return true
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

	if event.ID == "" {
		c.Logger.Warn().Str("type", string(event.Type)).Msg("Stripe webhook missing event ID; ignoring")
		return ResponseData{StatusCode: http.StatusOK}
	}

	shouldProcess, err := beginStripeWebhookEvent(c, c.Conn, &event)
	if err != nil {
		c.Logger.Error().Err(err).Str("eventID", event.ID).Msg("failed to initialize Stripe webhook event state")
		return ResponseData{StatusCode: http.StatusOK}
	}
	if !shouldProcess {
		c.Logger.Info().Str("eventID", event.ID).Str("type", string(event.Type)).Msg("already processed Stripe webhook event; ignoring")
		return ResponseData{StatusCode: http.StatusOK}
	}
	markFailed := func(processErr error) {
		if processErr == nil {
			return
		}
		if err := finishStripeWebhookEvent(c, c.Conn, &event, processErr); err != nil {
			c.Logger.Error().Err(err).Str("eventID", event.ID).Msg("failed to mark Stripe webhook event failure state")
		}
	}

	c.Logger.Info().Str("type", string(event.Type)).Msg("received Stripe webhook")

	sc := stripe.NewClient(config.Config.Stripe.SecretKey)

	if isMembershipGracePaymentRetryEvent(&event) {
		if checkMembershipEventOrder(c, &event) {
			handleMembershipGracePaymentRetryWebhook(c, sc, &event)
		}
	}

	if isMembershipPaymentIntentEvent(&event) && !checkMembershipEventOrder(c, &event) {
		if err := finishStripeWebhookEvent(c, c.Conn, &event, nil); err != nil {
			c.Logger.Error().Err(err).Str("eventID", event.ID).Msg("failed to mark Stripe webhook event processed")
		}
		return ResponseData{StatusCode: http.StatusOK}
	}
	if tryHandleMembershipPaymentIntentWebhook(c, sc, &event) {
		if err := finishStripeWebhookEvent(c, c.Conn, &event, nil); err != nil {
			c.Logger.Error().Err(err).Str("eventID", event.ID).Msg("failed to mark Stripe webhook event processed")
		}
		return ResponseData{StatusCode: http.StatusOK}
	}

	priceIDs, err := stripePriceIDsForEvent(c, sc, &event)
	if err != nil {
		markFailed(err)
		c.Logger.Error().Err(err).Str("type", string(event.Type)).Msg("failed to resolve price IDs for stripe event")
		return ResponseData{StatusCode: http.StatusOK}
	}

	kind, err := classifyStripePriceIDs(c, c.Conn, priceIDs)
	if err != nil {
		markFailed(err)
		c.Logger.Error().Err(err).Msg("failed to classify stripe webhook by price")
		return ResponseData{StatusCode: http.StatusOK}
	}

	switch kind {
	case stripeWebhookKindTicket:
		res := handleTicketStripeEvent(c, sc, &event)
		if res.StatusCode >= http.StatusBadRequest {
			markFailed(oops.New(nil, "ticket Stripe webhook handler returned status %d", res.StatusCode))
			return res
		}
		if err := finishStripeWebhookEvent(c, c.Conn, &event, nil); err != nil {
			c.Logger.Error().Err(err).Str("eventID", event.ID).Msg("failed to mark Stripe webhook event processed")
		}
		return res
	case stripeWebhookKindMembership:
		if checkMembershipEventOrder(c, &event) {
			handleMembershipStripeEvent(c, sc, &event)
		}
		if err := finishStripeWebhookEvent(c, c.Conn, &event, nil); err != nil {
			c.Logger.Error().Err(err).Str("eventID", event.ID).Msg("failed to mark Stripe webhook event processed")
		}
		return ResponseData{StatusCode: http.StatusOK}
	default:
		c.Logger.Warn().
			Str("type", string(event.Type)).
			Strs("prices", priceIDs).
			Msg("Stripe webhook did not match any known ticket or membership price; ignoring")
		if err := finishStripeWebhookEvent(c, c.Conn, &event, nil); err != nil {
			c.Logger.Error().Err(err).Str("eventID", event.ID).Msg("failed to mark Stripe webhook event processed")
		}
		return ResponseData{StatusCode: http.StatusOK}
	}
}

func finishStripeWebhookEvent(ctx context.Context, conn db.ConnOrTx, event *stripe.Event, processErr error) error {
	if processErr == nil {
		_, err := conn.Exec(ctx, `
			UPDATE stripe_webhook_event
			SET
				status = 'processed',
				last_error = NULL,
				updated_at = NOW(),
				processed_at = NOW()
			WHERE event_id = $1
		`, event.ID)
		if err != nil {
			return oops.New(err, "failed to mark stripe webhook event as processed")
		}
		return nil
	}

	_, err := conn.Exec(ctx, `
		UPDATE stripe_webhook_event
		SET
			status = 'failed',
			last_error = $2,
			updated_at = NOW()
		WHERE event_id = $1
	`, event.ID, processErr.Error())
	if err != nil {
		return oops.New(err, "failed to mark stripe webhook event as failed")
	}
	return nil
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

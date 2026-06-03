package website

import (
	"context"
	"encoding/json"

	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/perf"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v84"
)

// ProcessMembershipStripeWebhookForTests routes a membership Stripe event through the
// same handlers used by StripeWebhook. Intended for admin subscription integration tests.
func ProcessMembershipStripeWebhookForTests(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client, event *stripe.Event) {
	if event == nil {
		return
	}

	logger := logging.GlobalLogger()
	c := &RequestContext{
		ctx:    ctx,
		Conn:   pool,
		Logger: logger,
		Perf:   perf.MakeNewRequestPerf("subscription-test", "POST", "/stripe/webhook"),
	}

	if isMembershipGracePaymentRetryEvent(event) {
		handleMembershipGracePaymentRetryWebhook(c, sc, event)
	}

	if tryHandleMembershipPaymentIntentWebhook(c, sc, event) {
		return
	}

	handleMembershipStripeEvent(c, sc, event)
}

// StripeEventFromObject builds a synthetic Stripe webhook event from a Stripe object.
func StripeEventFromObject(eventType stripe.EventType, obj any) (*stripe.Event, error) {
	raw, err := json.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return &stripe.Event{
		Type: eventType,
		Data: &stripe.EventData{Raw: raw},
	}, nil
}

// InvoicePaymentIntentForTests resolves the payment intent on an invoice. Exported for admin tests.
func InvoicePaymentIntentForTests(ctx context.Context, sc *stripe.Client, inv *stripe.Invoice) (*stripe.PaymentIntent, string, error) {
	return invoicePaymentIntent(ctx, sc, inv)
}

// Exported for admin subscription integration tests.
func RetrySubscriptionPaymentForTests(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client, customerID string) error {
	logger := logging.GlobalLogger()
	c := &RequestContext{
		ctx:    ctx,
		Conn:   pool,
		Logger: logger,
		Perf:   perf.MakeNewRequestPerf("subscription-test", "POST", "/stripe/webhook"),
	}
	maybeRetrySubscriptionPaymentForCustomer(c, sc, customerID)
	return nil
}

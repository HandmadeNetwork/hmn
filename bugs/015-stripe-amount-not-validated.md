# Stripe ticket confirmation doesn't validate amount or payment status

**File:** `src/website/tickets.go:402-432`, `src/website/stripe.go:61-86`
**Severity:** Medium — money path; Stripe signature verification bounds most attack surface, but logic gaps remain
**Status:** Partially confirmed — worth a second pair of eyes

## The situation

`StripeWebhook` verifies the webhook signature (`stripe.go:28-37`), so the event is trusted to come from Stripe. That blocks forged webhook bodies. Good. But once past the signature check, `confirmStripeTicketPurchase` performs only two sanity checks:

```go
if ticket.ID.String() != session.ClientReferenceID { /* error */ }
if ticket.StripeCheckoutSessionID != session.ID    { /* error */ }
```

…and then marks the ticket paid:

```go
_, err := conn.Exec(ctx,
    `UPDATE ticket SET pending = FALSE, stripe_pi_id = $2,
                       stripe_price_amount = $3, stripe_price_currency = $4
     WHERE id = $1`,
    ticket.ID, session.PaymentIntent.ID,
    fmt.Sprintf("%d", session.AmountTotal), session.Currency,
)
```

The missing checks:

### 1. `session.PaymentStatus` is not verified

Stripe's `checkout.session.completed` event fires for both `"paid"` and `"unpaid"` checkout sessions — the latter happens for delayed-payment methods (SEPA, OXXO) where Stripe emits `completed` at confirmation time and the actual money lands minutes to days later. With `ExcludedPaymentMethodTypes: []*string{stripe.String("us_bank_account")}` at `tickets.go:369` the current purchase flow excludes ACH, so the main offender is blocked. But any Stripe dashboard change that adds back a delayed payment method (or a future event config) will quietly ship unpaid tickets as paid.

Standard Stripe guidance: check `session.PaymentStatus == "paid"` before fulfillment. There is also `"no_payment_required"`, which is legal for zero-amount sessions — relevant only if anyone ever creates a free-ticket price.

### 2. `session.AmountTotal` is not checked against the expected price

The ticket is created with `Price: stripe.String(metadata.StripePriceID)` at `tickets.go:357`. The webhook receives `session.AmountTotal` and writes it to the DB, but does not compare it to the expected amount for that price ID. Stripe's Checkout Session Price object is server-side, so a user on the browser cannot change it — that bounds the attack surface to "someone changes the price ID in Stripe dashboard between ticket creation and webhook", which is an insider threat, not a web-user threat.

More realistically: coupon / promo code. If Stripe ever has a promo code that applies to this price, `AmountTotal` drops and the ticket still gets confirmed. Today, `CheckoutSessionParams` at `tickets.go:353-378` does not set `AllowPromotionCodes`, so the default is off — check this in the Stripe dashboard to confirm.

### 3. `session.PaymentIntent.ID` is dereferenced without a nil check

```go
ticket.ID, session.PaymentIntent.ID, ...
```

For delayed-payment or zero-amount sessions, `PaymentIntent` can be nil. Same dereference in the logging branch at `stripe.go:82-83`:

```go
Str("payment intent ID", session.PaymentIntent.ID).
```

If `PaymentIntent` is ever nil on an incoming session, both of these nil-deref and crash the webhook handler. The webhook signature check won't catch it — Stripe sends valid events with nil PaymentIntent.

## Suggested hardening

```go
func confirmStripeTicketPurchase(...) error {
    // Existing sanity checks ...

    if session.PaymentStatus != stripe.CheckoutSessionPaymentStatusPaid {
        return oops.New(nil, "ticket session not paid: status=%s", session.PaymentStatus)
    }
    if session.PaymentIntent == nil {
        return oops.New(nil, "ticket session has no payment intent")
    }

    // Fetch the expected amount from config and compare
    event, ok := hmndata.FindTicketEventBySlug(ticket.EventSlug)
    if !ok {
        return oops.New(nil, "no event config for slug %s", ticket.EventSlug)
    }
    if session.AmountTotal != event.ExpectedAmountTotal {
        return oops.New(nil, "amount mismatch: got %d, expected %d",
            session.AmountTotal, event.ExpectedAmountTotal)
    }
    // ... existing UPDATE ...
}
```

("ExpectedAmountTotal" as a hypothetical field — whatever the right config surface is.)

This file is tagged "Critical" severity in the audit because it's the money path. Likelihood is Low because the Stripe SDK takes care of the hardest parts. But every one of these gaps is worth closing before someone flips an unrelated Stripe dashboard toggle and turns one into an incident.

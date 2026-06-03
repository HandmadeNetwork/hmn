package website

import (
	"encoding/json"
	"strconv"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/stripe/stripe-go/v84"
)

func tryHandleMembershipPaymentIntentWebhook(c *RequestContext, sc *stripe.Client, event *stripe.Event) bool {
	switch event.Type {
	case stripe.EventTypePaymentIntentProcessing,
		stripe.EventTypePaymentIntentRequiresAction,
		stripe.EventTypePaymentIntentPaymentFailed:
	default:
		return false
	}

	var pi stripe.PaymentIntent
	if err := json.Unmarshal(event.Data.Raw, &pi); err != nil {
		c.Logger.Error().Err(err).Str("type", string(event.Type)).Msg("failed to unmarshal payment_intent for membership")
		return false
	}

	return handleMembershipPaymentIntentWebhook(c, sc, event.Type, &pi)
}

func handleMembershipPaymentIntentWebhook(c *RequestContext, sc *stripe.Client, eventType stripe.EventType, pi *stripe.PaymentIntent) bool {
	if pi == nil || pi.Customer == nil {
		return false
	}

	user, err := db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE stripe_customer_id = $1", pi.Customer.ID)
	if err != nil {
		return false
	}
	if user.StripeSubscriptionID == nil {
		return false
	}

	fullPI, err := retrievePaymentIntent(c, sc, pi.ID)
	if err != nil {
		logging.Warn().Err(err).Str("paymentIntentID", pi.ID).Msg("failed to retrieve payment intent for membership webhook")
		fullPI = pi
	} else if fullPI != nil {
		pi = fullPI
	}

	pmType := paymentIntentPaymentMethodType(c, sc, pi)
	now := SubscriptionNow()

	switch eventType {
	case stripe.EventTypePaymentIntentProcessing, stripe.EventTypePaymentIntentRequiresAction:
		if shouldGrantGraceForPaymentIntent(pi, pmType) && canStartGrace(user, now) {
			shouldSendVerificationEmail := shouldSendACHVerificationEmailForPaymentIntent(pi, pmType)
			if user.StripeCustomerID == nil || user.StripeSubscriptionID == nil {
				return true
			}
			invoiceID, invoiceErr := findOpenSubscriptionInvoice(c, sc, *user.StripeCustomerID, *user.StripeSubscriptionID)
			if invoiceErr != nil {
				logging.Warn().Err(invoiceErr).Int("userID", user.ID).Msg("failed to resolve open subscription invoice for payment intent webhook")
				return true
			}
			if invoiceID == "" {
				logging.Info().Int("userID", user.ID).Str("eventType", string(eventType)).Msg("skipping grace start; no open subscription invoice")
				return true
			}
			startedGrace, err := startGracePeriod(c, c.Conn, user.ID, now)
			if err != nil {
				logging.Error().Err(err).Int("userID", user.ID).Msg("failed to start grace period from payment intent webhook")
			} else if startedGrace && shouldSendVerificationEmail {
				sendACHVerificationGraceEmail(c, user.ID)
			}
		}
	case stripe.EventTypePaymentIntentPaymentFailed:
		if paymentIntentIsHardDecline(pi, pmType) {
			if shouldStartGraceOnPaymentFailure(user, now, false) {
				if _, err := startGracePeriod(c, c.Conn, user.ID, now); err != nil {
					logging.Error().Err(err).Int("userID", user.ID).Msg("failed to start grace period from payment_intent.payment_failed")
				}
			} else if err := revokeSubscriptionAccessAfterDeclinedPayment(c, c.Conn, user.ID, "incomplete"); err != nil {
				logging.Error().Err(err).Int("userID", user.ID).Msg("failed to revoke access from payment_intent.payment_failed")
			}
		}
	default:
		return false
	}

	return true
}

func handleCheckoutAsyncPaymentFailed(c *RequestContext, sc *stripe.Client, session *stripe.CheckoutSession) {
	if session.ClientReferenceID == "" {
		return
	}
	userID, err := strconv.Atoi(session.ClientReferenceID)
	if err != nil {
		return
	}
	if err := revokeSubscriptionAccessAfterDeclinedPayment(c, c.Conn, userID, "incomplete"); err != nil {
		logging.Error().Err(err).Int("userID", userID).Msg("failed to revoke access after async checkout payment failed")
	}
}

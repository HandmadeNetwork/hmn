package website

import (
	"encoding/json"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/stripe/stripe-go/v84"
)

func isMembershipGracePaymentRetryEvent(event *stripe.Event) bool {
	switch event.Type {
	case "payment_method.attached", "customer.updated":
		return true
	default:
		return false
	}
}

func handleMembershipGracePaymentRetryWebhook(c *RequestContext, sc *stripe.Client, event *stripe.Event) {
	switch event.Type {
	case "payment_method.attached":
		var pm stripe.PaymentMethod
		if err := json.Unmarshal(event.Data.Raw, &pm); err != nil {
			c.Logger.Error().Err(err).Msg("failed to unmarshal payment_method.attached for grace retry")
			return
		}
		if pm.Customer == nil {
			return
		}
		maybeRetrySubscriptionPaymentForCustomer(c, sc, pm.Customer.ID)
	case "customer.updated":
		if !customerDefaultPaymentMethodChanged(event) {
			return
		}
		var customer stripe.Customer
		if err := json.Unmarshal(event.Data.Raw, &customer); err != nil {
			c.Logger.Error().Err(err).Msg("failed to unmarshal customer.updated for grace retry")
			return
		}
		maybeRetrySubscriptionPaymentForCustomer(c, sc, customer.ID)
	}
}

func customerDefaultPaymentMethodChanged(event *stripe.Event) bool {
	if event.Data == nil || len(event.Data.PreviousAttributes) == 0 {
		return false
	}
	if _, ok := event.Data.PreviousAttributes["invoice_settings"]; ok {
		return true
	}
	if _, ok := event.Data.PreviousAttributes["default_source"]; ok {
		return true
	}
	return false
}

func maybeRetrySubscriptionPaymentForCustomer(c *RequestContext, sc *stripe.Client, customerID string) {
	user, err := db.QueryOne[models.User](c, c.Conn, "SELECT $columns FROM hmn_user WHERE stripe_customer_id = $1", customerID)
	if err != nil {
		return
	}
	if err := retryPastDueSubscriptionPayment(c, c.Conn, sc, user); err != nil {
		logging.Warn().Err(err).Int("userID", user.ID).Str("customerID", customerID).Msg("failed to retry subscription payment after payment method change")
	}
}

package website

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/models"
	"github.com/stripe/stripe-go/v84"
)

func isAsyncPaymentMethodType(pmType string) bool {
	switch pmType {
	case "us_bank_account", "acss_debit", "sepa_debit":
		return true
	default:
		return false
	}
}

func paymentIntentHasMicrodepositVerification(pi *stripe.PaymentIntent) bool {
	if pi == nil || pi.NextAction == nil {
		return false
	}
	return pi.NextAction.Type == stripe.PaymentIntentNextActionTypeVerifyWithMicrodeposits
}

// shouldGrantGraceForPaymentIntent returns true when payment is in-flight for an async
// method (e.g. ACH processing or microdeposit verification), not a card decline.
func shouldGrantGraceForPaymentIntent(pi *stripe.PaymentIntent, paymentMethodType string) bool {
	if pi == nil {
		return false
	}
	switch pi.Status {
	case stripe.PaymentIntentStatusRequiresAction:
		// Bank microdeposit verification; payment method type is often unset on the PI this early.
		return paymentIntentHasMicrodepositVerification(pi)
	case stripe.PaymentIntentStatusProcessing:
		return isAsyncPaymentMethodType(resolvePaymentMethodType(pi, paymentMethodType))
	default:
		return false
	}
}

func resolvePaymentMethodType(pi *stripe.PaymentIntent, explicit string) string {
	if explicit != "" {
		return explicit
	}
	if pi.PaymentMethod != nil && pi.PaymentMethod.Type != "" {
		return string(pi.PaymentMethod.Type)
	}
	for _, t := range pi.PaymentMethodTypes {
		if isAsyncPaymentMethodType(t) {
			return t
		}
	}
	return ""
}

func paymentIntentIsHardDecline(pi *stripe.PaymentIntent, paymentMethodType string) bool {
	if pi == nil || isAsyncPaymentMethodType(paymentMethodType) {
		return false
	}
	if pi.LastPaymentError != nil {
		return isHardDeclineErrorCode(string(pi.LastPaymentError.Code))
	}
	return pi.Status == stripe.PaymentIntentStatusRequiresPaymentMethod ||
		pi.Status == stripe.PaymentIntentStatusCanceled
}

func isHardDeclineErrorCode(code string) bool {
	switch stripe.ErrorCode(code) {
	case stripe.ErrorCodeCardDeclined,
		stripe.ErrorCodeInsufficientFunds,
		stripe.ErrorCodeExpiredCard,
		stripe.ErrorCodeIncorrectCVC,
		stripe.ErrorCodeIncorrectNumber,
		stripe.ErrorCodeInvalidCVC,
		stripe.ErrorCodeInvalidExpiryMonth,
		stripe.ErrorCodeInvalidExpiryYear,
		stripe.ErrorCodeInvalidNumber,
		stripe.ErrorCodeProcessingError,
		stripe.ErrorCodeAuthenticationRequired:
		return true
	default:
		return false
	}
}

func paymentIntentPaymentMethodType(ctx context.Context, sc *stripe.Client, pi *stripe.PaymentIntent) string {
	if pi == nil {
		return ""
	}
	if resolved := resolvePaymentMethodType(pi, ""); resolved != "" {
		return resolved
	}
	if pi.PaymentMethod == nil || pi.PaymentMethod.ID == "" {
		return ""
	}
	pm, err := sc.V1PaymentMethods.Retrieve(ctx, pi.PaymentMethod.ID, nil)
	if err != nil || pm == nil {
		return ""
	}
	return string(pm.Type)
}

func retrievePaymentIntent(ctx context.Context, sc *stripe.Client, paymentIntentID string) (*stripe.PaymentIntent, error) {
	if paymentIntentID == "" {
		return nil, nil
	}
	params := &stripe.PaymentIntentRetrieveParams{}
	params.AddExpand("payment_method")
	return sc.V1PaymentIntents.Retrieve(ctx, paymentIntentID, params)
}

func checkoutSessionPaymentIntent(ctx context.Context, sc *stripe.Client, session *stripe.CheckoutSession) (*stripe.PaymentIntent, string, error) {
	if session == nil || session.PaymentIntent == nil {
		return nil, "", nil
	}
	piID := session.PaymentIntent.ID
	pi, err := retrievePaymentIntent(ctx, sc, piID)
	if err != nil {
		return nil, "", err
	}
	return pi, paymentIntentPaymentMethodType(ctx, sc, pi), nil
}

func invoicePaymentIntent(ctx context.Context, sc *stripe.Client, inv *stripe.Invoice) (*stripe.PaymentIntent, string, error) {
	if inv == nil {
		return nil, "", nil
	}
	params := &stripe.InvoicePaymentListParams{
		Invoice: stripe.String(inv.ID),
	}
	params.AddExpand("data.payment.payment_intent")

	var pi *stripe.PaymentIntent
	sc.V1InvoicePayments.List(ctx, params)(func(ip *stripe.InvoicePayment, err error) bool {
		if err != nil || ip == nil || ip.Payment == nil || ip.Payment.PaymentIntent == nil {
			return true
		}
		pi = ip.Payment.PaymentIntent
		return false
	})
	if pi == nil {
		return nil, "", nil
	}
	if pi.PaymentMethod == nil || pi.PaymentMethod.Type == "" {
		full, err := retrievePaymentIntent(ctx, sc, pi.ID)
		if err != nil {
			return nil, "", err
		}
		pi = full
	}
	return pi, paymentIntentPaymentMethodType(ctx, sc, pi), nil
}

func shouldGrantGraceForSubscription(ctx context.Context, sc *stripe.Client, sub *stripe.Subscription) bool {
	if sub == nil || sub.LatestInvoice == nil {
		return false
	}
	invParams := &stripe.InvoiceRetrieveParams{}
	invParams.AddExpand("payments.data.payment.payment_intent")
	inv, err := sc.V1Invoices.Retrieve(ctx, sub.LatestInvoice.ID, invParams)
	if err != nil {
		return false
	}
	pi, pmType, err := invoicePaymentIntent(ctx, sc, inv)
	if err != nil {
		return false
	}
	return shouldGrantGraceForPaymentIntent(pi, pmType)
}

func shouldGrantGraceForInvoice(ctx context.Context, sc *stripe.Client, inv *stripe.Invoice) bool {
	pi, pmType, err := invoicePaymentIntent(ctx, sc, inv)
	if err != nil {
		return false
	}
	return shouldGrantGraceForPaymentIntent(pi, pmType)
}

func invoicePaymentIsHardDecline(ctx context.Context, sc *stripe.Client, inv *stripe.Invoice) bool {
	pi, pmType, err := invoicePaymentIntent(ctx, sc, inv)
	if err != nil || pi == nil {
		return false
	}
	return paymentIntentIsHardDecline(pi, pmType)
}

// shouldStartGraceOnPaymentFailure returns true when a failed payment should begin the
// one-time grace period. Async methods (ACH processing / verification) always qualify;
// card declines qualify only for existing subscribers (renewal), not initial sign-up.
func shouldStartGraceOnPaymentFailure(user *models.User, now time.Time, asyncGraceEligible bool) bool {
	if user == nil || !canStartGrace(user, now) {
		return false
	}
	if asyncGraceEligible {
		return true
	}
	return user.IsSubscribed
}

package website

import (
	"testing"

	"github.com/stripe/stripe-go/v84"
	"github.com/stretchr/testify/assert"
)

func TestShouldGrantGraceForPaymentIntent(t *testing.T) {
	achPI := &stripe.PaymentIntent{Status: stripe.PaymentIntentStatusProcessing}
	assert.True(t, shouldGrantGraceForPaymentIntent(achPI, "us_bank_account"))
	assert.False(t, shouldGrantGraceForPaymentIntent(achPI, "card"))

	cardPI := &stripe.PaymentIntent{Status: stripe.PaymentIntentStatusProcessing}
	assert.False(t, shouldGrantGraceForPaymentIntent(cardPI, "card"))

	achVerify := &stripe.PaymentIntent{
		Status: stripe.PaymentIntentStatusRequiresAction,
		NextAction: &stripe.PaymentIntentNextAction{
			Type: stripe.PaymentIntentNextActionTypeVerifyWithMicrodeposits,
		},
	}
	assert.True(t, shouldGrantGraceForPaymentIntent(achVerify, "us_bank_account"))
}

func TestPaymentIntentIsHardDecline(t *testing.T) {
	declined := &stripe.PaymentIntent{
		Status: stripe.PaymentIntentStatusRequiresPaymentMethod,
		LastPaymentError: &stripe.Error{
			Code: stripe.ErrorCodeInsufficientFunds,
		},
	}
	assert.True(t, paymentIntentIsHardDecline(declined, "card"))
	assert.False(t, paymentIntentIsHardDecline(declined, "us_bank_account"))

	processingACH := &stripe.PaymentIntent{Status: stripe.PaymentIntentStatusProcessing}
	assert.False(t, paymentIntentIsHardDecline(processingACH, "us_bank_account"))
}

func TestIsAsyncPaymentMethodType(t *testing.T) {
	assert.True(t, isAsyncPaymentMethodType("us_bank_account"))
	assert.False(t, isAsyncPaymentMethodType("card"))
}

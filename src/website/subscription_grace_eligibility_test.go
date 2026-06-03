package website

import (
	"testing"
	"time"

	"github.com/stripe/stripe-go/v84"
	"github.com/stretchr/testify/assert"

	"git.handmade.network/hmn/hmn/src/models"
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
	assert.True(t, shouldGrantGraceForPaymentIntent(achVerify, ""))

	cardVerify := &stripe.PaymentIntent{
		Status: stripe.PaymentIntentStatusRequiresAction,
		NextAction: &stripe.PaymentIntentNextAction{
			Type: stripe.PaymentIntentNextActionTypeUseStripeSDK,
		},
	}
	assert.False(t, shouldGrantGraceForPaymentIntent(cardVerify, "card"))
}

func TestResolvePaymentMethodType(t *testing.T) {
	pi := &stripe.PaymentIntent{
		PaymentMethodTypes: []string{"card", "us_bank_account"},
	}
	assert.Equal(t, "us_bank_account", resolvePaymentMethodType(pi, ""))
	assert.Equal(t, "card", resolvePaymentMethodType(pi, "card"))
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

func TestShouldStartGraceOnPaymentFailure(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	activeSubscriber := &models.User{
		IsSubscribed:     true,
		GraceAvailable:   true,
	}
	assert.True(t, shouldStartGraceOnPaymentFailure(activeSubscriber, now, false))
	assert.True(t, shouldStartGraceOnPaymentFailure(activeSubscriber, now, true))

	initialSignup := &models.User{
		IsSubscribed:   false,
		GraceAvailable: true,
	}
	assert.False(t, shouldStartGraceOnPaymentFailure(initialSignup, now, false))
	assert.True(t, shouldStartGraceOnPaymentFailure(initialSignup, now, true))

	noGraceLeft := &models.User{
		IsSubscribed:   true,
		GraceAvailable: false,
	}
	assert.False(t, shouldStartGraceOnPaymentFailure(noGraceLeft, now, false))
}

package website

import (
	"testing"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/stretchr/testify/assert"
)

func statusPtr(s string) *string {
	return &s
}

func timePtr(t time.Time) *time.Time {
	return &t
}

func TestSubscriptionNowUsesOverride(t *testing.T) {
	original := config.Config.Stripe.SubscriptionNowOverride
	defer func() {
		config.Config.Stripe.SubscriptionNowOverride = original
		ClearSubscriptionNowForTests()
	}()

	fixed := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	config.Config.Stripe.SubscriptionNowOverride = fixed.Format(time.RFC3339)
	ClearSubscriptionNowForTests()

	assert.Equal(t, fixed, SubscriptionNow())

	SetSubscriptionNowForTests(fixed.Add(2 * time.Hour))
	assert.Equal(t, fixed.Add(2*time.Hour), SubscriptionNow())
}

func TestIsGraceActive(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)
	endsAt := now.Add(48 * time.Hour)

	user := &models.User{
		SubscriptionStatus:   statusPtr(SubscriptionStatusGracePeriod),
		GracePeriodStartedAt: timePtr(now.Add(-24 * time.Hour)),
		GracePeriodEndsAt:    timePtr(endsAt),
	}
	assert.True(t, isGraceActive(user, now))
	assert.False(t, isGraceActive(user, endsAt))
	assert.False(t, isGraceActive(&models.User{}, now))
}

func TestCanStartGrace(t *testing.T) {
	now := time.Date(2026, 5, 24, 12, 0, 0, 0, time.UTC)

	assert.True(t, canStartGrace(&models.User{GraceAvailable: true}, now))
	assert.False(t, canStartGrace(&models.User{GraceAvailable: false}, now))

	activeGraceUser := &models.User{
		GraceAvailable:       true,
		SubscriptionStatus:   statusPtr(SubscriptionStatusGracePeriod),
		GracePeriodStartedAt: timePtr(now.Add(-24 * time.Hour)),
		GracePeriodEndsAt:    timePtr(now.Add(24 * time.Hour)),
	}
	assert.False(t, canStartGrace(activeGraceUser, now))
}

func TestIsFailedPaymentStripeStatus(t *testing.T) {
	assert.True(t, isFailedPaymentStripeStatus("past_due"))
	assert.True(t, isFailedPaymentStripeStatus("unpaid"))
	assert.False(t, isFailedPaymentStripeStatus("active"))
	assert.False(t, isFailedPaymentStripeStatus("trialing"))
}

func TestStripeSubscriptionGrantsAccess(t *testing.T) {
	assert.True(t, stripeSubscriptionGrantsAccess("active"))
	assert.True(t, stripeSubscriptionGrantsAccess("trialing"))
	assert.False(t, stripeSubscriptionGrantsAccess("past_due"))
}

func TestUserInGracePeriod(t *testing.T) {
	assert.True(t, userInGracePeriod(&models.User{SubscriptionStatus: statusPtr(SubscriptionStatusGracePeriod)}))
	assert.False(t, userInGracePeriod(&models.User{SubscriptionStatus: statusPtr("active")}))
}

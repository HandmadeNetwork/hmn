package website

import (
	"testing"

	"git.handmade.network/hmn/hmn/src/hmnurl"
	"github.com/stretchr/testify/assert"
)

// Routing regression: info page and subscription manage must not share the same path.
func TestFoundationMembershipVsManageRoutes(t *testing.T) {
	assert.True(t, hmnurl.RegexHSFMembership.MatchString("/foundation/membership"))
	assert.False(t, hmnurl.RegexHSFMembership.MatchString("/foundation/membership/manage"))

	assert.True(t, hmnurl.RegexSubscriptionManage.MatchString("/foundation/membership/manage"))
	assert.False(t, hmnurl.RegexSubscriptionManage.MatchString("/foundation/membership"))
}

func TestStripeWebhookIsTheOnlyStripeEndpoint(t *testing.T) {
	assert.True(t, hmnurl.RegexStripeWebhook.MatchString("/stripe/webhook"))
}

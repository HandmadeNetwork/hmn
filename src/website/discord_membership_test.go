package website

import (
	"testing"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/stretchr/testify/assert"
)

func TestUserEligibleForSupporterDiscordRole(t *testing.T) {
	assert.False(t, userEligibleForSupporterDiscordRole(nil))
	assert.False(t, userEligibleForSupporterDiscordRole(&models.User{IsSubscribed: false}))
	assert.True(t, userEligibleForSupporterDiscordRole(&models.User{IsSubscribed: true}))
}

func TestUserNeedsDiscordLinkReminder(t *testing.T) {
	assert.False(t, userNeedsDiscordLinkReminder(nil))
	assert.False(t, userNeedsDiscordLinkReminder(&models.User{IsSubscribed: false}))
	assert.False(t, userNeedsDiscordLinkReminder(&models.User{
		IsSubscribed: true,
		DiscordUser:  &models.DiscordUser{},
	}))
	assert.True(t, userNeedsDiscordLinkReminder(&models.User{IsSubscribed: true}))
	assert.False(t, userNeedsDiscordLinkReminder(&models.User{
		IsSubscribed:                         true,
		DismissedMembershipDiscordLinkBanner: true,
	}))
}

func TestSyncSupporterDiscordRoleNoOpsWithoutConfig(t *testing.T) {
	original := config.Config.Discord.SupporterRoleID
	config.Config.Discord.SupporterRoleID = ""
	defer func() {
		config.Config.Discord.SupporterRoleID = original
	}()

	// Should return without panicking when role ID is unset.
	SyncSupporterDiscordRole(t.Context(), nil, 1)
}

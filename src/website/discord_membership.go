package website

import (
	"context"
	"errors"
	"net/http"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

func userEligibleForSupporterDiscordRole(user *models.User) bool {
	return user != nil && user.IsSubscribed
}

func userNeedsDiscordLinkReminder(user *models.User) bool {
	return user != nil &&
		user.IsSubscribed &&
		user.DiscordUser == nil &&
		!user.DismissedMembershipDiscordLinkBanner
}

func SyncSupporterDiscordRole(ctx context.Context, conn db.ConnOrTx, userID int) {
	roleID := config.Config.Discord.SupporterRoleID
	if roleID == "" {
		return
	}

	user, err := db.QueryOne[models.User](ctx, conn, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
	if err != nil {
		if err != db.NotFound {
			logging.Warn().Err(err).Int("userID", userID).Msg("failed to load user for supporter Discord role sync")
		}
		return
	}

	discordUser, err := db.QueryOne[models.DiscordUser](ctx, conn,
		"SELECT $columns FROM discord_user WHERE hmn_user_id = $1",
		userID,
	)
	if err == db.NotFound {
		return
	}
	if err != nil {
		logging.Warn().Err(err).Int("userID", userID).Msg("failed to load Discord user for supporter role sync")
		return
	}

	syncSupporterDiscordRoleForUser(ctx, user, discordUser.UserID, roleID)
}

func SyncSupporterDiscordRoleForCustomer(ctx context.Context, conn db.ConnOrTx, stripeCustomerID string) {
	if config.Config.Discord.SupporterRoleID == "" {
		return
	}

	user, err := db.QueryOne[models.User](ctx, conn,
		"SELECT $columns FROM hmn_user WHERE stripe_customer_id = $1",
		stripeCustomerID,
	)
	if err != nil {
		if err != db.NotFound {
			logging.Warn().Err(err).Str("customerID", stripeCustomerID).Msg("failed to load user for supporter Discord role sync")
		}
		return
	}

	SyncSupporterDiscordRole(ctx, conn, user.ID)
}

func syncSupporterDiscordRoleForUser(ctx context.Context, user *models.User, discordUserID, roleID string) {
	var err error
	if userEligibleForSupporterDiscordRole(user) {
		err = discord.AddGuildMemberRole(ctx, discordUserID, roleID)
	} else {
		err = discord.RemoveGuildMemberRole(ctx, discordUserID, roleID)
	}

	if err == nil {
		return
	}
	if errors.Is(err, discord.NotFound) {
		logging.Warn().
			Int("userID", user.ID).
			Str("discordUserID", discordUserID).
			Bool("grant", userEligibleForSupporterDiscordRole(user)).
			Msg("Discord user not in guild; skipped supporter role sync")
		return
	}
	logging.Warn().
		Err(err).
		Int("userID", user.ID).
		Str("discordUserID", discordUserID).
		Bool("grant", userEligibleForSupporterDiscordRole(user)).
		Msg("failed to sync supporter Discord role")
}

func DismissMembershipDiscordLinkBanner(c *RequestContext) ResponseData {
	_, err := c.Conn.Exec(c,
		`UPDATE hmn_user SET dismissed_membership_discord_link_banner = true WHERE id = $1`,
		c.CurrentUser.ID,
	)
	if err != nil {
		return c.JSONErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to dismiss membership Discord link banner"))
	}

	return c.JSONResponse(http.StatusOK, map[string]any{"success": true})
}

package website

import (
	"context"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
)

// RevokeSubscriptionAccessAfterDeclinedPayment clears member access when a payment
// was declined (not processing). Exported for admin subscription test tooling.
func RevokeSubscriptionAccessAfterDeclinedPayment(ctx context.Context, conn db.ConnOrTx, userID int, subscriptionStatus string) error {
	return revokeSubscriptionAccessAfterDeclinedPayment(ctx, conn, userID, subscriptionStatus)
}

// revokeSubscriptionAccessAfterDeclinedPayment clears member access when a payment
// was declined (not processing). Restores grace_available so a future ACH attempt can
// still use the one-time grace period.
func revokeSubscriptionAccessAfterDeclinedPayment(ctx context.Context, conn db.ConnOrTx, userID int, subscriptionStatus string) error {
	_, err := conn.Exec(ctx, `
		UPDATE hmn_user
		SET
			is_subscribed = false,
			subscription_status = $1,
			grace_period_started_at = NULL,
			grace_period_ends_at = NULL,
			grace_available = true
		WHERE id = $2
	`, subscriptionStatus, userID)
	if err != nil {
		return err
	}
	logging.Info().Int("userID", userID).Str("status", subscriptionStatus).Msg("revoked subscription access after declined payment")
	SyncSupporterDiscordRole(ctx, conn, userID)
	return nil
}

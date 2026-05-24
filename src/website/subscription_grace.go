package website

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
)

const (
	SubscriptionStatusGracePeriod         = "grace_period"
	SubscriptionStatusGraceFailed           = "grace_failed"
	SubscriptionStatusPendingVerification   = "pending_verification"
	subscriptionGracePeriodDuration         = 7 * 24 * time.Hour
)

var subscriptionNowOverride *time.Time

func SubscriptionNow() time.Time {
	if subscriptionNowOverride != nil {
		return *subscriptionNowOverride
	}
	if override := config.Config.Stripe.SubscriptionNowOverride; override != "" {
		if parsed, err := time.Parse(time.RFC3339, override); err == nil {
			return parsed
		}
	}
	return time.Now()
}

func SetSubscriptionNowForTests(t time.Time) {
	subscriptionNowOverride = &t
}

func ClearSubscriptionNowForTests() {
	subscriptionNowOverride = nil
}

func isGraceActive(user *models.User, now time.Time) bool {
	if user == nil || user.GracePeriodEndsAt == nil {
		return false
	}
	if user.SubscriptionStatus != nil && *user.SubscriptionStatus == SubscriptionStatusGracePeriod {
		return now.Before(*user.GracePeriodEndsAt)
	}
	return user.GracePeriodStartedAt != nil && now.Before(*user.GracePeriodEndsAt)
}

func canStartGrace(user *models.User, now time.Time) bool {
	if user == nil || !user.GraceAvailable {
		return false
	}
	if isGraceActive(user, now) {
		return false
	}
	return true
}

func stripeSubscriptionGrantsAccess(status string) bool {
	return status == "active" || status == "trialing"
}

func isFailedPaymentStripeStatus(status string) bool {
	switch status {
	case "past_due", "unpaid", "incomplete", "incomplete_expired":
		return true
	default:
		return false
	}
}

func startGracePeriod(ctx context.Context, conn db.ConnOrTx, userID int, now time.Time) error {
	endsAt := now.Add(subscriptionGracePeriodDuration)
	_, err := conn.Exec(ctx, `
		UPDATE hmn_user
		SET
			is_subscribed = true,
			subscription_status = $1,
			grace_period_started_at = $2,
			grace_period_ends_at = $3,
			grace_available = false
		WHERE id = $4
	`, SubscriptionStatusGracePeriod, now, endsAt, userID)
	if err != nil {
		return err
	}
	logging.Info().Int("userID", userID).Time("graceEndsAt", endsAt).Msg("started subscription grace period")
	return nil
}

func clearGracePeriod(ctx context.Context, conn db.ConnOrTx, userID int) error {
	_, err := conn.Exec(ctx, `
		UPDATE hmn_user
		SET
			grace_period_started_at = NULL,
			grace_period_ends_at = NULL,
			grace_available = true
		WHERE id = $1
	`, userID)
	if err != nil {
		return err
	}
	logging.Info().Int("userID", userID).Msg("cleared subscription grace period after successful payment")
	return nil
}

func expireGracePeriod(ctx context.Context, conn db.ConnOrTx, userID int) error {
	_, err := conn.Exec(ctx, `
		UPDATE hmn_user
		SET
			is_subscribed = false,
			subscription_status = $1,
			grace_period_started_at = NULL,
			grace_period_ends_at = NULL,
			grace_available = false
		WHERE id = $2
		  AND subscription_status = $3
	`, SubscriptionStatusGraceFailed, userID, SubscriptionStatusGracePeriod)
	if err != nil {
		return err
	}
	logging.Info().Int("userID", userID).Msg("expired subscription grace period without payment")
	return nil
}

func expireDueGracePeriods(ctx context.Context, conn db.ConnOrTx, now time.Time) (int64, error) {
	tag, err := conn.Exec(ctx, `
		UPDATE hmn_user
		SET
			is_subscribed = false,
			subscription_status = $1,
			grace_period_started_at = NULL,
			grace_period_ends_at = NULL,
			grace_available = false
		WHERE subscription_status = $2
		  AND grace_period_ends_at IS NOT NULL
		  AND grace_period_ends_at < $3
	`, SubscriptionStatusGraceFailed, SubscriptionStatusGracePeriod, now)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

func userInGracePeriod(user *models.User) bool {
	return user != nil && user.SubscriptionStatus != nil && *user.SubscriptionStatus == SubscriptionStatusGracePeriod
}

func StartSubscriptionGracePeriod(ctx context.Context, conn db.ConnOrTx, userID int) error {
	return startGracePeriod(ctx, conn, userID, SubscriptionNow())
}

func ExpireSubscriptionGracePeriods(ctx context.Context, conn db.ConnOrTx) (int64, error) {
	return expireDueGracePeriods(ctx, conn, SubscriptionNow())
}

package website

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/stripe/stripe-go/v84"
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
	SyncSupporterDiscordRole(ctx, conn, userID)
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

func activateSubscriptionAfterSuccessfulPayment(ctx context.Context, conn db.ConnOrTx, userID int, currentPeriodEnd *time.Time) error {
	if err := clearGracePeriod(ctx, conn, userID); err != nil {
		return err
	}
	if currentPeriodEnd != nil {
		_, err := conn.Exec(ctx, `
			UPDATE hmn_user
			SET is_subscribed = true, subscription_status = 'active', current_period_end = $1
			WHERE id = $2
		`, currentPeriodEnd, userID)
		return err
	}
	_, err := conn.Exec(ctx, `
		UPDATE hmn_user
		SET is_subscribed = true, subscription_status = 'active'
		WHERE id = $1
	`, userID)
	return err
}

func subscriptionIDFromInvoice(inv *stripe.Invoice) string {
	if inv == nil {
		return ""
	}
	if inv.Lines != nil {
		for _, line := range inv.Lines.Data {
			if line.Subscription != nil && line.Subscription.ID != "" {
				return line.Subscription.ID
			}
		}
	}
	if inv.Parent != nil && inv.Parent.SubscriptionDetails != nil && inv.Parent.SubscriptionDetails.Subscription != nil {
		return inv.Parent.SubscriptionDetails.Subscription.ID
	}
	return ""
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
	SyncSupporterDiscordRole(ctx, conn, userID)
	return nil
}

func expireDueGracePeriods(ctx context.Context, conn db.ConnOrTx, now time.Time) ([]int, error) {
	userIDPtrs, err := db.Query[int](ctx, conn, `
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
		RETURNING id
	`, SubscriptionStatusGraceFailed, SubscriptionStatusGracePeriod, now)
	if err != nil {
		return nil, err
	}
	userIDs := make([]int, len(userIDPtrs))
	for i, id := range userIDPtrs {
		userIDs[i] = *id
	}
	return userIDs, nil
}

func userInGracePeriod(user *models.User) bool {
	return user != nil && user.SubscriptionStatus != nil && *user.SubscriptionStatus == SubscriptionStatusGracePeriod
}

func userNeedsBankVerificationReminder(user *models.User) bool {
	if user == nil || user.SubscriptionStatus == nil {
		return false
	}
	switch *user.SubscriptionStatus {
	case SubscriptionStatusPendingVerification, "incomplete":
		return true
	case SubscriptionStatusGracePeriod:
		return user.IsSubscribed
	default:
		return false
	}
}

func gracePeriodDaysRemaining(user *models.User, now time.Time) int {
	if user == nil || user.GracePeriodEndsAt == nil || !user.GracePeriodEndsAt.After(now) {
		return 0
	}

	hoursRemaining := user.GracePeriodEndsAt.Sub(now).Hours()
	days := int(hoursRemaining / 24)
	if hoursRemaining > float64(days*24) {
		days++
	}
	if days < 1 {
		return 1
	}
	return days
}

func StartSubscriptionGracePeriod(ctx context.Context, conn db.ConnOrTx, userID int) error {
	return startGracePeriod(ctx, conn, userID, SubscriptionNow())
}

func ExpireSubscriptionGracePeriods(ctx context.Context, conn db.ConnOrTx) (int64, error) {
	userIDs, err := expireDueGracePeriods(ctx, conn, SubscriptionNow())
	if err != nil {
		return 0, err
	}
	for _, userID := range userIDs {
		SyncSupporterDiscordRole(ctx, conn, userID)
	}
	return int64(len(userIDs)), nil
}

func shouldRetrySubscriptionPayment(user *models.User) bool {
	if user == nil || user.StripeCustomerID == nil || user.StripeSubscriptionID == nil {
		return false
	}
	if userInGracePeriod(user) {
		return true
	}
	if !user.IsSubscribed || user.SubscriptionStatus == nil {
		return false
	}
	switch *user.SubscriptionStatus {
	case SubscriptionStatusPendingVerification, "incomplete", "past_due", "unpaid":
		return true
	default:
		return false
	}
}

func retryPastDueSubscriptionPayment(ctx context.Context, conn db.ConnOrTx, sc *stripe.Client, user *models.User) error {
	if !shouldRetrySubscriptionPayment(user) {
		return nil
	}

	invoiceID, err := findOpenSubscriptionInvoice(ctx, sc, *user.StripeCustomerID, *user.StripeSubscriptionID)
	if err != nil {
		return err
	}
	if invoiceID == "" {
		logging.Info().Int("userID", user.ID).Msg("no open subscription invoice to retry")
		return nil
	}

	inv, err := sc.V1Invoices.Pay(ctx, invoiceID, &stripe.InvoicePayParams{})
	if err != nil {
		return err
	}

	logging.Info().Int("userID", user.ID).Str("invoiceID", invoiceID).Str("status", string(inv.Status)).Msg("retried open subscription invoice payment")

	if inv.Status == stripe.InvoiceStatusPaid {
		var renewalDate *time.Time
		if user.StripeSubscriptionID != nil {
			if sub, retrieveErr := sc.V1Subscriptions.Retrieve(ctx, *user.StripeSubscriptionID, nil); retrieveErr == nil {
				renewalDate = getSubscriptionPeriodEnd(sub)
			}
		}
		if err := activateSubscriptionAfterSuccessfulPayment(ctx, conn, user.ID, renewalDate); err != nil {
			return err
		}
		SyncSupporterDiscordRole(ctx, conn, user.ID)
	}

	return nil
}

func findOpenSubscriptionInvoice(ctx context.Context, sc *stripe.Client, customerID, subscriptionID string) (string, error) {
	subParams := &stripe.SubscriptionRetrieveParams{}
	subParams.AddExpand("latest_invoice")
	sub, err := sc.V1Subscriptions.Retrieve(ctx, subscriptionID, subParams)
	if err != nil {
		return "", err
	}
	if sub.LatestInvoice != nil && sub.LatestInvoice.Status == stripe.InvoiceStatusOpen && sub.LatestInvoice.AmountRemaining > 0 {
		return sub.LatestInvoice.ID, nil
	}

	listParams := &stripe.InvoiceListParams{
		Customer: stripe.String(customerID),
		Status:   stripe.String(string(stripe.InvoiceStatusOpen)),
	}
	var invoiceID string
	var listErr error
	sc.V1Invoices.List(ctx, listParams)(func(inv *stripe.Invoice, err error) bool {
		if err != nil {
			listErr = err
			return false
		}
		if inv == nil || inv.AmountRemaining <= 0 {
			return true
		}
		if !invoiceBelongsToSubscription(inv, subscriptionID) {
			return true
		}
		invoiceID = inv.ID
		return false
	})
	if listErr != nil {
		return "", listErr
	}
	return invoiceID, nil
}

func invoiceBelongsToSubscription(inv *stripe.Invoice, subscriptionID string) bool {
	if inv == nil || inv.Parent == nil || inv.Parent.SubscriptionDetails == nil {
		return false
	}
	sub := inv.Parent.SubscriptionDetails.Subscription
	return sub != nil && sub.ID == subscriptionID
}

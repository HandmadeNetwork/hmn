package admintools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v84"
)

type achTestSetup struct {
	userID          int
	customerID      string
	paymentMethodID string
	testClockID     string
}

func runCardOrACHScenario(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client, scenario subscriptionTestScenario) (subscriptionTestResult, error) {
	sctx := newScenarioCtx(scenario.Name, 6)

	username := fmt.Sprintf("subtest_%s", uuid.NewString()[:8])
	var userID int
	var emailAddress string
	if err := sctx.step(fmt.Sprintf("Creating test user: %s", username), func() error {
		userID, emailAddress = createSubscriptionTestUser(ctx, pool, username)
		sctx.printf("user_id=%d email=%s\n", userID, emailAddress)
		return nil
	}); err != nil {
		return subscriptionTestResultPass, err
	}

	var customer *stripe.Customer
	if err := sctx.step("Creating Stripe customer", func() error {
		customerParams := &stripe.CustomerCreateParams{
			Email: stripe.String(emailAddress),
			Name:  stripe.String(username),
			Metadata: map[string]string{
				"user_id": strconv.Itoa(userID),
			},
		}
		if testClockID := config.Config.Stripe.TestClockID; testClockID != "" {
			customerParams.TestClock = stripe.String(testClockID)
		}
		var err error
		customer, err = sc.V1Customers.Create(ctx, customerParams)
		if err != nil {
			return err
		}
		sctx.printf("customer_id=%s\n", customer.ID)
		return nil
	}); err != nil {
		return subscriptionTestResultPass, err
	}

	var paymentMethod *stripe.PaymentMethod
	if err := sctx.step(fmt.Sprintf("Creating payment method (%s)", scenario.Name), func() error {
		var err error
		paymentMethod, err = scenario.CreatePaymentMethod(ctx, sc)
		if err != nil {
			return err
		}
		sctx.printf("payment_method_id=%s\n", paymentMethod.ID)
		return nil
	}); err != nil {
		return subscriptionTestResultPass, err
	}

	var attachErr error
	if err := sctx.step("Attaching payment method and creating membership", func() error {
		_, attachErr = sc.V1PaymentMethods.Attach(ctx, paymentMethod.ID, &stripe.PaymentMethodAttachParams{
			Customer: stripe.String(customer.ID),
		})
		if attachErr != nil && isExpectedACHVerificationPending(attachErr) {
			sctx.printf("ACH verification is pending; membership will complete after verification.\n")
			if updateErr := persistPendingVerificationState(ctx, pool, userID, customer.ID); updateErr != nil {
				return updateErr
			}
			printSubscriptionData(ctx, pool, userID)
			return nil
		}
		return attachErr
	}); err != nil {
		return subscriptionTestResultPass, err
	}
	if attachErr != nil {
		if isExpectedACHVerificationPending(attachErr) {
			return subscriptionTestResultPending, nil
		}
	}

	return completeSubscription(ctx, pool, sc, userID, customer.ID, paymentMethod.ID)
}

func runDeclinedCardScenario(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client) (subscriptionTestResult, error) {
	username := fmt.Sprintf("subtest_%s", uuid.NewString()[:8])
	fmt.Printf("[1/6] Creating test user: %s\n", username)
	userID, emailAddress := createSubscriptionTestUser(ctx, pool, username)
	fmt.Printf("      user_id=%d email=%s\n", userID, emailAddress)

	fmt.Printf("[2/6] Creating Stripe customer\n")
	customerParams := &stripe.CustomerCreateParams{
		Email: stripe.String(emailAddress),
		Name:  stripe.String(username),
		Metadata: map[string]string{
			"user_id": strconv.Itoa(userID),
		},
	}
	if testClockID := config.Config.Stripe.TestClockID; testClockID != "" {
		customerParams.TestClock = stripe.String(testClockID)
	}
	customer, err := sc.V1Customers.Create(ctx, customerParams)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      customer_id=%s\n", customer.ID)

	fmt.Printf("[3/6] Creating declined card payment method (tok_chargeDeclined)\n")
	paymentMethod, err := sc.V1PaymentMethods.Create(ctx, &stripe.PaymentMethodCreateParams{
		Type: stripe.String("card"),
		Card: &stripe.PaymentMethodCreateCardParams{
			Token: stripe.String("tok_chargeDeclined"),
		},
	})
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      payment_method_id=%s\n", paymentMethod.ID)

	fmt.Printf("[4/6] Attaching payment method and creating membership\n")
	_, attachErr := sc.V1PaymentMethods.Attach(ctx, paymentMethod.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customer.ID),
	})
	if attachErr != nil && !isStripeCardDeclined(attachErr) {
		return subscriptionTestResultPass, attachErr
	}
	if attachErr != nil {
		fmt.Printf("      card declined during payment method attach (expected)\n")
	}

	var subscriptionID string
	stripeStatus := "incomplete"

	if attachErr == nil {
		subscriptionParams := &stripe.SubscriptionCreateParams{
			Customer:             stripe.String(customer.ID),
			DefaultPaymentMethod: stripe.String(paymentMethod.ID),
			CollectionMethod:     stripe.String("charge_automatically"),
			PaymentBehavior:      stripe.String("allow_incomplete"),
			Items: []*stripe.SubscriptionCreateItemParams{
				{Price: stripe.String(config.Config.Stripe.PriceID)},
			},
			Metadata: map[string]string{
				"user_id": strconv.Itoa(userID),
			},
		}
		subscriptionParams.AddExpand("latest_invoice.payments.data.payment.payment_intent")

		subscription, createErr := sc.V1Subscriptions.Create(ctx, subscriptionParams)
		if createErr != nil && !isStripeCardDeclined(createErr) {
			return subscriptionTestResultPass, createErr
		}
		if createErr != nil {
			fmt.Printf("      card declined during membership create (expected)\n")
		} else {
			fmt.Printf("      membership_subscription_id=%s status=%s\n", subscription.ID, subscription.Status)
			if subscription.Status == stripe.SubscriptionStatusActive || subscription.Status == stripe.SubscriptionStatusTrialing {
				return subscriptionTestResultPass, fmt.Errorf("expected membership subscription to fail payment, got status=%s", subscription.Status)
			}
			subscriptionID = subscription.ID
			stripeStatus = string(subscription.Status)
		}
	}

	fmt.Printf("[5/6] Simulating declined payment access revoke\n")
	if err := website.RevokeSubscriptionAccessAfterDeclinedPayment(ctx, pool, userID, stripeStatus); err != nil {
		return subscriptionTestResultPass, err
	}
	if subscriptionID != "" {
		_, err = pool.Exec(ctx, `
			UPDATE hmn_user
			SET stripe_customer_id = $1, stripe_subscription_id = $2, cancel_at_period_end = false
			WHERE id = $3
		`, customer.ID, subscriptionID, userID)
	} else {
		_, err = pool.Exec(ctx, `
			UPDATE hmn_user
			SET stripe_customer_id = $1, cancel_at_period_end = false
			WHERE id = $2
		`, customer.ID, userID)
	}
	if err != nil {
		return subscriptionTestResultPass, err
	}

	fmt.Printf("[6/6] Verifying stored membership data after decline\n")
	user, err := db.QueryOne[models.User](ctx, pool, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if user.IsSubscribed {
		return subscriptionTestResultPass, fmt.Errorf("expected is_subscribed=false after card decline")
	}
	if user.SubscriptionStatus == nil || *user.SubscriptionStatus != stripeStatus {
		return subscriptionTestResultPass, fmt.Errorf("expected membership subscription_status=%s, got %s", stripeStatus, stringOrEmpty(user.SubscriptionStatus))
	}
	if !user.GraceAvailable {
		return subscriptionTestResultPass, fmt.Errorf("expected grace_available=true after card decline (grace not consumed)")
	}
	if user.GracePeriodStartedAt != nil || user.GracePeriodEndsAt != nil {
		return subscriptionTestResultPass, fmt.Errorf("expected no grace period after card decline")
	}

	payments, err := db.Query[models.UserPayment](ctx, pool, `
		SELECT $columns FROM user_payment WHERE user_id = $1
	`, userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if len(payments) > 0 {
		return subscriptionTestResultPass, fmt.Errorf("expected no paid invoices after card decline, got %d payment rows", len(payments))
	}

	printSubscriptionData(ctx, pool, userID)
	return subscriptionTestResultPass, nil
}

func runEuroCardChargeScenario(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client) (subscriptionTestResult, error) {
	sctx := newScenarioCtx("Credit card one-time charge (EUR)", 5)

	username := fmt.Sprintf("subtest_%s", uuid.NewString()[:8])
	var userID int
	var emailAddress string
	if err := sctx.step(fmt.Sprintf("Creating test user for EUR charge: %s", username), func() error {
		userID, emailAddress = createSubscriptionTestUser(ctx, pool, username)
		sctx.printf("user_id=%d email=%s\n", userID, emailAddress)
		return nil
	}); err != nil {
		return subscriptionTestResultPass, err
	}

	var customer *stripe.Customer
	if err := sctx.step("Creating Stripe customer", func() error {
		customerParams := &stripe.CustomerCreateParams{
			Email: stripe.String(emailAddress),
			Name:  stripe.String(username),
			Metadata: map[string]string{
				"user_id": strconv.Itoa(userID),
			},
		}
		if testClockID := config.Config.Stripe.TestClockID; testClockID != "" {
			customerParams.TestClock = stripe.String(testClockID)
		}
		var err error
		customer, err = sc.V1Customers.Create(ctx, customerParams)
		if err != nil {
			return err
		}
		sctx.printf("customer_id=%s\n", customer.ID)
		return nil
	}); err != nil {
		return subscriptionTestResultPass, err
	}

	var pm *stripe.PaymentMethod
	if err := sctx.step("Creating and attaching tok_visa payment method", func() error {
		var err error
		pm, err = createCardPaymentMethod(ctx, sc, "tok_visa")
		if err != nil {
			return err
		}
		_, err = sc.V1PaymentMethods.Attach(ctx, pm.ID, &stripe.PaymentMethodAttachParams{
			Customer: stripe.String(customer.ID),
		})
		return err
	}); err != nil {
		return subscriptionTestResultPass, err
	}

	var pi *stripe.PaymentIntent
	if err := sctx.step("Creating one-time EUR card charge", func() error {
		var err error
		pi, err = sc.V1PaymentIntents.Create(ctx, &stripe.PaymentIntentCreateParams{
			Amount:        stripe.Int64(500), // EUR 5.00
			Currency:      stripe.String("eur"),
			Customer:      stripe.String(customer.ID),
			PaymentMethod: stripe.String(pm.ID),
			PaymentMethodTypes: []*string{
				stripe.String("card"),
			},
			Confirm:     stripe.Bool(true),
			Description: stripe.String("HMN admin membership test EUR card charge"),
		})
		if err != nil {
			return err
		}
		sctx.printf("payment_intent_id=%s status=%s amount=%d currency=%s\n", pi.ID, pi.Status, pi.Amount, pi.Currency)
		return nil
	}); err != nil {
		return subscriptionTestResultPass, err
	}

	if err := sctx.step("Verifying one-time EUR charge success", func() error {
		if pi.Currency != stripe.CurrencyEUR {
			return fmt.Errorf("expected currency=eur, got %s", pi.Currency)
		}
		if pi.Status != stripe.PaymentIntentStatusSucceeded {
			return fmt.Errorf("expected payment_intent status=succeeded, got %s", pi.Status)
		}
		return nil
	}); err != nil {
		return subscriptionTestResultPass, err
	}

	return subscriptionTestResultPass, nil
}

func runACHGraceExpiryScenario(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client) (subscriptionTestResult, error) {
	defer website.ClearSubscriptionNowForTests()

	fmt.Printf("[1/7] Creating Stripe test clock\n")
	testClock, err := createTestClock(ctx, sc, "ach-grace-expiry")
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      test_clock_id=%s frozen_time=%s\n", testClock.ID, time.Unix(testClock.FrozenTime, 0).UTC().Format(time.RFC3339))
	defer deleteTestClock(ctx, sc, testClock.ID)

	setup, err := setupACHPendingOnClock(ctx, pool, sc, testClock.ID)
	if err != nil {
		return subscriptionTestResultPass, err
	}

	fmt.Printf("[5/7] Starting grace period (simulating payment failure while ACH verification is pending)\n")
	syncSubscriptionNowToTestClock(ctx, sc, testClock.ID)
	if err := website.StartSubscriptionGracePeriod(ctx, pool, setup.userID); err != nil {
		return subscriptionTestResultPass, err
	}

	fmt.Printf("[6/7] Advancing test clock by 14 days (past 7-day grace period)\n")
	clockTime, err := advanceTestClockBy(ctx, sc, testClock.ID, 14*24*time.Hour)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      clock frozen_time=%s\n", clockTime.UTC().Format(time.RFC3339))
	website.SetSubscriptionNowForTests(clockTime)

	fmt.Printf("[7/7] Expiring due grace periods and verifying final state\n")
	expiredCount, err := website.ExpireSubscriptionGracePeriods(ctx, pool)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      expired grace periods: %d\n", expiredCount)

	user, err := db.QueryOne[models.User](ctx, pool, "SELECT $columns FROM hmn_user WHERE id = $1", setup.userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if user.IsSubscribed {
		return subscriptionTestResultPass, fmt.Errorf("expected is_subscribed=false after grace expiry")
	}
	if user.SubscriptionStatus == nil || *user.SubscriptionStatus != website.SubscriptionStatusGraceFailed {
		return subscriptionTestResultPass, fmt.Errorf("expected membership subscription_status=%s, got %s", website.SubscriptionStatusGraceFailed, stringOrEmpty(user.SubscriptionStatus))
	}
	if user.GraceAvailable {
		return subscriptionTestResultPass, fmt.Errorf("expected grace_available=false after grace expiry")
	}

	printSubscriptionData(ctx, pool, setup.userID)
	return subscriptionTestResultPass, nil
}

func runACHVerificationAfterAdvanceScenario(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client) (subscriptionTestResult, error) {
	defer website.ClearSubscriptionNowForTests()

	fmt.Printf("[1/8] Creating Stripe test clock\n")
	testClock, err := createTestClock(ctx, sc, "ach-verify-after-advance")
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      test_clock_id=%s frozen_time=%s\n", testClock.ID, time.Unix(testClock.FrozenTime, 0).UTC().Format(time.RFC3339))
	defer deleteTestClock(ctx, sc, testClock.ID)

	setup, err := setupACHPendingOnClock(ctx, pool, sc, testClock.ID)
	if err != nil {
		return subscriptionTestResultPass, err
	}

	fmt.Printf("[5/8] Advancing test clock by 2 days (simulating microdeposit wait)\n")
	clockTime, err := advanceTestClockBy(ctx, sc, testClock.ID, 2*24*time.Hour)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      clock frozen_time=%s\n", clockTime.UTC().Format(time.RFC3339))
	website.SetSubscriptionNowForTests(clockTime)

	fmt.Printf("[6/8] Triggering ACH verification via SetupIntent\n")
	if err := verifyACHPaymentMethod(ctx, sc, setup.customerID, setup.paymentMethodID); err != nil {
		return subscriptionTestResultPass, err
	}

	fmt.Printf("[7/8] Attaching verified payment method and creating membership\n")
	_, err = sc.V1PaymentMethods.Attach(ctx, setup.paymentMethodID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(setup.customerID),
	})
	if err != nil {
		return subscriptionTestResultPass, fmt.Errorf("attach verified ACH payment method: %w", err)
	}

	result, err := completeSubscription(ctx, pool, sc, setup.userID, setup.customerID, setup.paymentMethodID)
	if err != nil {
		return result, err
	}

	fmt.Printf("[8/8] Verifying membership is active after ACH verification\n")
	user, err := db.QueryOne[models.User](ctx, pool, "SELECT $columns FROM hmn_user WHERE id = $1", setup.userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if !user.IsSubscribed {
		return subscriptionTestResultPass, fmt.Errorf("expected is_subscribed=true after ACH verification")
	}
	if user.SubscriptionStatus == nil || *user.SubscriptionStatus != "active" {
		return subscriptionTestResultPass, fmt.Errorf("expected membership subscription_status=active, got %s", stringOrEmpty(user.SubscriptionStatus))
	}

	return result, nil
}

func runCardRenewalFailureGraceRecoveryScenario(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client) (subscriptionTestResult, error) {
	defer website.ClearSubscriptionNowForTests()

	fmt.Printf("[1/10] Creating Stripe test clock\n")
	testClock, err := createTestClock(ctx, sc, "card-renewal-grace-recovery")
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      test_clock_id=%s frozen_time=%s\n", testClock.ID, time.Unix(testClock.FrozenTime, 0).UTC().Format(time.RFC3339))
	defer deleteTestClock(ctx, sc, testClock.ID)

	username := fmt.Sprintf("subtest_%s", uuid.NewString()[:8])
	fmt.Printf("[2/10] Creating test user: %s\n", username)
	userID, emailAddress := createSubscriptionTestUser(ctx, pool, username)
	fmt.Printf("      user_id=%d email=%s\n", userID, emailAddress)

	fmt.Printf("[3/10] Creating Stripe customer on test clock\n")
	customer, err := sc.V1Customers.Create(ctx, &stripe.CustomerCreateParams{
		Email:     stripe.String(emailAddress),
		Name:      stripe.String(username),
		TestClock: stripe.String(testClock.ID),
		Metadata: map[string]string{
			"user_id": strconv.Itoa(userID),
		},
	})
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      customer_id=%s\n", customer.ID)

	fmt.Printf("[4/10] Creating membership with tok_visa\n")
	visaPM, err := createCardPaymentMethod(ctx, sc, "tok_visa")
	if err != nil {
		return subscriptionTestResultPass, err
	}
	_, err = sc.V1PaymentMethods.Attach(ctx, visaPM.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customer.ID),
	})
	if err != nil {
		return subscriptionTestResultPass, err
	}
	_, err = sc.V1Customers.Update(ctx, customer.ID, &stripe.CustomerUpdateParams{
		InvoiceSettings: &stripe.CustomerUpdateInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(visaPM.ID),
		},
	})
	if err != nil {
		return subscriptionTestResultPass, err
	}

	result, err := completeSubscription(ctx, pool, sc, userID, customer.ID, visaPM.ID)
	if err != nil {
		return result, err
	}

	user, err := db.QueryOne[models.User](ctx, pool, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if !user.IsSubscribed || user.SubscriptionStatus == nil || *user.SubscriptionStatus != "active" {
		return subscriptionTestResultPass, fmt.Errorf("expected active membership before renewal, got is_subscribed=%v status=%s", user.IsSubscribed, stringOrEmpty(user.SubscriptionStatus))
	}
	subscriptionID := *user.StripeSubscriptionID

	subParams := &stripe.SubscriptionRetrieveParams{}
	subParams.AddExpand("items")
	subscription, err := sc.V1Subscriptions.Retrieve(ctx, subscriptionID, subParams)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if subscription.Items == nil || len(subscription.Items.Data) == 0 {
		return subscriptionTestResultPass, fmt.Errorf("membership subscription has no items")
	}
	periodEnd := subscription.Items.Data[0].CurrentPeriodEnd

	fmt.Printf("[5/10] Swapping default payment method to tok_chargeCustomerFail (fails on charge)\n")
	failPM, err := createCardPaymentMethod(ctx, sc, "tok_chargeCustomerFail")
	if err != nil {
		return subscriptionTestResultPass, err
	}
	_, err = sc.V1PaymentMethods.Attach(ctx, failPM.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customer.ID),
	})
	if err != nil {
		return subscriptionTestResultPass, fmt.Errorf("attach failing card: %w", err)
	}
	if err := setDefaultPaymentMethod(ctx, sc, customer.ID, subscriptionID, failPM.ID); err != nil {
		return subscriptionTestResultPass, err
	}

	fmt.Printf("[6/10] Advancing test clock past billing period end\n")
	targetTime := time.Unix(periodEnd, 0).Add(time.Hour)
	clockTime, err := advanceTestClockTo(ctx, sc, testClock.ID, targetTime)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      clock frozen_time=%s (period_end was %s)\n",
		clockTime.UTC().Format(time.RFC3339),
		time.Unix(periodEnd, 0).UTC().Format(time.RFC3339))
	website.SetSubscriptionNowForTests(clockTime)

	fmt.Printf("[7/10] Waiting for renewal payment attempt to fail\n")
	subscription, err = waitForSubscriptionStatus(ctx, sc, subscriptionID, "past_due", "unpaid", "incomplete")
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      membership subscription status=%s\n", subscription.Status)

	invParams := &stripe.InvoiceRetrieveParams{}
	invParams.AddExpand("payments.data.payment.payment_intent")
	failedInvoice, err := retrieveLatestSubscriptionInvoice(ctx, sc, subscription)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if failedInvoice == nil {
		return subscriptionTestResultPass, fmt.Errorf("expected open renewal invoice after failed payment")
	}
	failedInvoice, err = sc.V1Invoices.Retrieve(ctx, failedInvoice.ID, invParams)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      renewal invoice_id=%s status=%s\n", failedInvoice.ID, failedInvoice.Status)

	fmt.Printf("[8/10] Processing renewal failure webhooks\n")
	// Stripe test clock may deliver real webhooks if `stripe listen` is running; reset to the
	// expected pre-grace subscriber state so this scenario exercises our handlers.
	_, err = pool.Exec(ctx, `
		UPDATE hmn_user
		SET
			is_subscribed = true,
			subscription_status = 'active',
			grace_available = true,
			grace_period_started_at = NULL,
			grace_period_ends_at = NULL
		WHERE id = $1
	`, userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}

	failedInvoice, err = sc.V1Invoices.Retrieve(ctx, failedInvoice.ID, invParams)
	if err != nil {
		return subscriptionTestResultPass, err
	}

	if err := dispatchMembershipWebhook(ctx, pool, sc, "invoice.payment_failed", failedInvoice); err != nil {
		return subscriptionTestResultPass, err
	}
	if pi, _, err := website.InvoicePaymentIntentForTests(ctx, sc, failedInvoice); err != nil {
		return subscriptionTestResultPass, err
	} else if pi != nil {
		if err := dispatchMembershipWebhook(ctx, pool, sc, "payment_intent.payment_failed", pi); err != nil {
			return subscriptionTestResultPass, err
		}
	}
	if err := dispatchMembershipWebhook(ctx, pool, sc, "customer.subscription.updated", subscription); err != nil {
		return subscriptionTestResultPass, err
	}

	user, err = db.QueryOne[models.User](ctx, pool, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if user.SubscriptionStatus == nil || *user.SubscriptionStatus != website.SubscriptionStatusGracePeriod {
		return subscriptionTestResultPass, fmt.Errorf("expected membership subscription_status=%s after renewal failure, got %s", website.SubscriptionStatusGracePeriod, stringOrEmpty(user.SubscriptionStatus))
	}
	if !user.IsSubscribed {
		return subscriptionTestResultPass, fmt.Errorf("expected is_subscribed=true during grace period")
	}
	if user.GraceAvailable {
		return subscriptionTestResultPass, fmt.Errorf("expected grace_available=false after grace started")
	}
	if user.GracePeriodStartedAt == nil || user.GracePeriodEndsAt == nil {
		return subscriptionTestResultPass, fmt.Errorf("expected grace period dates to be set")
	}
	fmt.Printf("      grace period started, ends %s\n", user.GracePeriodEndsAt.UTC().Format(time.RFC3339))

	fmt.Printf("[9/10] Updating payment method to tok_visa and retrying payment\n")
	recoveryPM, err := createCardPaymentMethod(ctx, sc, "tok_visa")
	if err != nil {
		return subscriptionTestResultPass, err
	}
	_, err = sc.V1PaymentMethods.Attach(ctx, recoveryPM.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customer.ID),
	})
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if err := setDefaultPaymentMethod(ctx, sc, customer.ID, subscriptionID, recoveryPM.ID); err != nil {
		return subscriptionTestResultPass, err
	}

	recoveryPMObj, err := sc.V1PaymentMethods.Retrieve(ctx, recoveryPM.ID, nil)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if err := dispatchMembershipWebhook(ctx, pool, sc, "payment_method.attached", recoveryPMObj); err != nil {
		return subscriptionTestResultPass, err
	}

	subscription, err = waitForSubscriptionStatus(ctx, sc, subscriptionID, "active", "trialing")
	if err != nil {
		// Retry may have paid the invoice without flipping status yet; process invoice.paid if present.
		fmt.Printf("      membership not active yet (%v); checking for paid invoice\n", err)
	}

	paidInvoice, err := retrieveLatestSubscriptionInvoice(ctx, sc, subscription)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if paidInvoice != nil && paidInvoice.Status == stripe.InvoiceStatusPaid {
		if err := dispatchMembershipWebhook(ctx, pool, sc, "invoice.paid", paidInvoice); err != nil {
			return subscriptionTestResultPass, err
		}
	}
	if subscription != nil {
		if err := dispatchMembershipWebhook(ctx, pool, sc, "customer.subscription.updated", subscription); err != nil {
			return subscriptionTestResultPass, err
		}
	}

	fmt.Printf("[10/10] Verifying membership reinstated\n")
	user, err = db.QueryOne[models.User](ctx, pool, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	if !user.IsSubscribed {
		return subscriptionTestResultPass, fmt.Errorf("expected is_subscribed=true after payment method update")
	}
	if user.SubscriptionStatus == nil || *user.SubscriptionStatus != "active" {
		return subscriptionTestResultPass, fmt.Errorf("expected membership subscription_status=active after recovery, got %s", stringOrEmpty(user.SubscriptionStatus))
	}
	if user.GracePeriodStartedAt != nil || user.GracePeriodEndsAt != nil {
		return subscriptionTestResultPass, fmt.Errorf("expected grace period cleared after successful payment")
	}
	if !user.GraceAvailable {
		return subscriptionTestResultPass, fmt.Errorf("expected grace_available=true after grace consumed and cleared")
	}

	printSubscriptionData(ctx, pool, userID)
	return subscriptionTestResultPass, nil
}

func createCardPaymentMethod(ctx context.Context, sc *stripe.Client, token string) (*stripe.PaymentMethod, error) {
	pm, err := sc.V1PaymentMethods.Create(ctx, &stripe.PaymentMethodCreateParams{
		Type: stripe.String("card"),
		Card: &stripe.PaymentMethodCreateCardParams{
			Token: stripe.String(token),
		},
	})
	if err != nil {
		return nil, err
	}
	fmt.Printf("      payment_method_id=%s token=%s\n", pm.ID, token)
	return pm, nil
}

func setDefaultPaymentMethod(ctx context.Context, sc *stripe.Client, customerID, subscriptionID, paymentMethodID string) error {
	_, err := sc.V1Customers.Update(ctx, customerID, &stripe.CustomerUpdateParams{
		InvoiceSettings: &stripe.CustomerUpdateInvoiceSettingsParams{
			DefaultPaymentMethod: stripe.String(paymentMethodID),
		},
	})
	if err != nil {
		return fmt.Errorf("update customer default payment method: %w", err)
	}
	_, err = sc.V1Subscriptions.Update(ctx, subscriptionID, &stripe.SubscriptionUpdateParams{
		DefaultPaymentMethod: stripe.String(paymentMethodID),
	})
	if err != nil {
		return fmt.Errorf("update membership default payment method: %w", err)
	}
	return nil
}

func advanceTestClockTo(ctx context.Context, sc *stripe.Client, testClockID string, target time.Time) (time.Time, error) {
	_, err := sc.V1TestHelpersTestClocks.Advance(ctx, testClockID, &stripe.TestHelpersTestClockAdvanceParams{
		FrozenTime: stripe.Int64(target.Unix()),
	})
	if err != nil {
		return time.Time{}, err
	}
	clock, err := waitForTestClockReady(ctx, sc, testClockID)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(clock.FrozenTime, 0), nil
}

func waitForSubscriptionStatus(ctx context.Context, sc *stripe.Client, subscriptionID string, statuses ...string) (*stripe.Subscription, error) {
	want := make(map[string]struct{}, len(statuses))
	for _, s := range statuses {
		want[s] = struct{}{}
	}

	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		sub, err := sc.V1Subscriptions.Retrieve(ctx, subscriptionID, nil)
		if err != nil {
			return nil, err
		}
		if _, ok := want[string(sub.Status)]; ok {
			return sub, nil
		}
		time.Sleep(2 * time.Second)
	}
	sub, err := sc.V1Subscriptions.Retrieve(ctx, subscriptionID, nil)
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("membership subscription %s did not reach status %v within timeout (last status=%s)", subscriptionID, statuses, sub.Status)
}

func retrieveLatestSubscriptionInvoice(ctx context.Context, sc *stripe.Client, sub *stripe.Subscription) (*stripe.Invoice, error) {
	if sub == nil {
		return nil, fmt.Errorf("membership subscription is nil")
	}
	subParams := &stripe.SubscriptionRetrieveParams{}
	subParams.AddExpand("latest_invoice")
	fresh, err := sc.V1Subscriptions.Retrieve(ctx, sub.ID, subParams)
	if err != nil {
		return nil, err
	}
	if fresh.LatestInvoice != nil && fresh.LatestInvoice.ID != "" {
		return sc.V1Invoices.Retrieve(ctx, fresh.LatestInvoice.ID, nil)
	}
	return nil, nil
}

func dispatchMembershipWebhook(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client, eventType string, obj any) error {
	event, err := website.StripeEventFromObject(stripe.EventType(eventType), obj)
	if err != nil {
		return fmt.Errorf("build stripe event %s: %w", eventType, err)
	}
	website.ProcessMembershipStripeWebhookForTests(ctx, pool, sc, event)
	return nil
}

func setupACHPendingOnClock(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client, testClockID string) (*achTestSetup, error) {
	username := fmt.Sprintf("subtest_%s", uuid.NewString()[:8])
	fmt.Printf("[2/7] Creating test user: %s\n", username)
	userID, emailAddress := createSubscriptionTestUser(ctx, pool, username)
	fmt.Printf("      user_id=%d email=%s\n", userID, emailAddress)

	fmt.Printf("[3/7] Creating Stripe customer on test clock\n")
	customer, err := sc.V1Customers.Create(ctx, &stripe.CustomerCreateParams{
		Email:     stripe.String(emailAddress),
		Name:      stripe.String(username),
		TestClock: stripe.String(testClockID),
		Metadata: map[string]string{
			"user_id": strconv.Itoa(userID),
		},
	})
	if err != nil {
		return nil, err
	}
	fmt.Printf("      customer_id=%s\n", customer.ID)

	fmt.Printf("[4/7] Creating ACH payment method and reaching pending verification\n")
	paymentMethod, err := createACHPaymentMethod(ctx, sc)
	if err != nil {
		return nil, err
	}
	fmt.Printf("      payment_method_id=%s\n", paymentMethod.ID)

	_, err = sc.V1PaymentMethods.Attach(ctx, paymentMethod.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customer.ID),
	})
	if err == nil {
		return nil, fmt.Errorf("expected ACH attach to require verification")
	}
	if !isExpectedACHVerificationPending(err) {
		return nil, fmt.Errorf("attach ACH payment method: %w", err)
	}
	fmt.Printf("      ACH verification is pending; membership will complete after verification.\n")
	if err := persistPendingVerificationState(ctx, pool, userID, customer.ID); err != nil {
		return nil, err
	}
	printSubscriptionData(ctx, pool, userID)

	return &achTestSetup{
		userID:          userID,
		customerID:      customer.ID,
		paymentMethodID: paymentMethod.ID,
		testClockID:     testClockID,
	}, nil
}

func createACHPaymentMethod(ctx context.Context, sc *stripe.Client) (*stripe.PaymentMethod, error) {
	return sc.V1PaymentMethods.Create(ctx, &stripe.PaymentMethodCreateParams{
		Type: stripe.String("us_bank_account"),
		USBankAccount: &stripe.PaymentMethodCreateUSBankAccountParams{
			AccountHolderType: stripe.String("individual"),
			AccountType:       stripe.String("checking"),
			RoutingNumber:     stripe.String("110000000"),
			AccountNumber:     stripe.String("000123456789"),
		},
		BillingDetails: &stripe.PaymentMethodCreateBillingDetailsParams{
			Name: stripe.String("HMN ACH Test User"),
		},
	})
}

func verifyACHPaymentMethod(ctx context.Context, sc *stripe.Client, customerID, paymentMethodID string) error {
	setupIntent, err := sc.V1SetupIntents.Create(ctx, &stripe.SetupIntentCreateParams{
		Customer:           stripe.String(customerID),
		PaymentMethod:      stripe.String(paymentMethodID),
		PaymentMethodTypes: []*string{stripe.String("us_bank_account")},
		Confirm:            stripe.Bool(true),
		MandateData: &stripe.SetupIntentCreateMandateDataParams{
			CustomerAcceptance: &stripe.SetupIntentCreateMandateDataCustomerAcceptanceParams{
				Type: stripe.String("online"),
				Online: &stripe.SetupIntentCreateMandateDataCustomerAcceptanceOnlineParams{
					IPAddress: stripe.String("127.0.0.1"),
					UserAgent: stripe.String("HMN Admin Subscription Test"),
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("confirm setup intent for ACH verification: %w", err)
	}
	fmt.Printf("      setup_intent_id=%s status=%s\n", setupIntent.ID, setupIntent.Status)

	if setupIntent.Status == stripe.SetupIntentStatusRequiresAction {
		setupIntent, err = sc.V1SetupIntents.VerifyMicrodeposits(ctx, setupIntent.ID, &stripe.SetupIntentVerifyMicrodepositsParams{
			Amounts: []*int64{stripe.Int64(32), stripe.Int64(45)},
		})
		if err != nil {
			setupIntent, err = sc.V1SetupIntents.VerifyMicrodeposits(ctx, setupIntent.ID, &stripe.SetupIntentVerifyMicrodepositsParams{
				DescriptorCode: stripe.String("SM11AA"),
			})
			if err != nil {
				return fmt.Errorf("verify ACH microdeposits: %w", err)
			}
		}
		fmt.Printf("      setup_intent_id=%s status=%s after verification\n", setupIntent.ID, setupIntent.Status)
	}

	if setupIntent.Status != stripe.SetupIntentStatusSucceeded {
		return fmt.Errorf("setup intent did not succeed: status=%s", setupIntent.Status)
	}
	return nil
}

func completeSubscription(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client, userID int, customerID, paymentMethodID string) (subscriptionTestResult, error) {
	subscriptionParams := &stripe.SubscriptionCreateParams{
		Customer:             stripe.String(customerID),
		DefaultPaymentMethod: stripe.String(paymentMethodID),
		CollectionMethod:     stripe.String("charge_automatically"),
		PaymentBehavior:      stripe.String("allow_incomplete"),
		Items: []*stripe.SubscriptionCreateItemParams{
			{Price: stripe.String(config.Config.Stripe.PriceID)},
		},
		Metadata: map[string]string{
			"user_id": strconv.Itoa(userID),
		},
	}
	subscriptionParams.AddExpand("latest_invoice")

	subscription, err := sc.V1Subscriptions.Create(ctx, subscriptionParams)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      membership_subscription_id=%s status=%s\n", subscription.ID, subscription.Status)

	fmt.Printf("[5/6] Writing membership state to database\n")
	renewalDate := getSubscriptionPeriodEndFromStripe(subscription)
	isSubscribed := subscription.Status == stripe.SubscriptionStatusActive || subscription.Status == stripe.SubscriptionStatusTrialing
	_, err = pool.Exec(ctx, `
		UPDATE hmn_user
		SET
			is_subscribed = $1,
			stripe_customer_id = $2,
			stripe_subscription_id = $3,
			subscription_status = $4,
			current_period_end = $5,
			cancel_at_period_end = $6
		WHERE id = $7
	`, isSubscribed, customerID, subscription.ID, subscription.Status, renewalDate, subscription.CancelAtPeriodEnd, userID)
	if err != nil {
		return subscriptionTestResultPass, err
	}

	var invoice *stripe.Invoice
	if subscription.LatestInvoice != nil && subscription.LatestInvoice.ID != "" {
		invoice, err = sc.V1Invoices.Retrieve(ctx, subscription.LatestInvoice.ID, nil)
		if err != nil {
			return subscriptionTestResultPass, err
		}
	}
	if invoice != nil && invoice.StatusTransitions != nil && invoice.StatusTransitions.PaidAt > 0 {
		paidAt := time.Unix(invoice.StatusTransitions.PaidAt, 0)
		_, err = pool.Exec(ctx, `
			INSERT INTO user_payment (user_id, stripe_invoice_id, amount_cents, currency, paid_at)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (stripe_invoice_id) DO UPDATE SET
				amount_cents = EXCLUDED.amount_cents,
				currency = EXCLUDED.currency,
				paid_at = EXCLUDED.paid_at
		`, userID, invoice.ID, invoice.AmountPaid, string(invoice.Currency), paidAt)
		if err != nil {
			return subscriptionTestResultPass, err
		}
	}

	fmt.Printf("[6/6] Verifying and printing stored membership data\n")
	if err := validateStoredSubscriptionData(ctx, pool, userID, customerID, subscription.ID); err != nil {
		return subscriptionTestResultPass, err
	}
	printSubscriptionData(ctx, pool, userID)
	return subscriptionTestResultPass, nil
}

func createTestClock(ctx context.Context, sc *stripe.Client, name string) (*stripe.TestHelpersTestClock, error) {
	return sc.V1TestHelpersTestClocks.Create(ctx, &stripe.TestHelpersTestClockCreateParams{
		FrozenTime: stripe.Int64(time.Now().Unix()),
		Name:       stripe.String(name),
	})
}

func advanceTestClockBy(ctx context.Context, sc *stripe.Client, testClockID string, duration time.Duration) (time.Time, error) {
	clock, err := sc.V1TestHelpersTestClocks.Retrieve(ctx, testClockID, nil)
	if err != nil {
		return time.Time{}, err
	}

	target := time.Unix(clock.FrozenTime, 0).Add(duration)
	_, err = sc.V1TestHelpersTestClocks.Advance(ctx, testClockID, &stripe.TestHelpersTestClockAdvanceParams{
		FrozenTime: stripe.Int64(target.Unix()),
	})
	if err != nil {
		return time.Time{}, err
	}

	clock, err = waitForTestClockReady(ctx, sc, testClockID)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(clock.FrozenTime, 0), nil
}

func waitForTestClockReady(ctx context.Context, sc *stripe.Client, testClockID string) (*stripe.TestHelpersTestClock, error) {
	deadline := time.Now().Add(2 * time.Minute)
	for time.Now().Before(deadline) {
		clock, err := sc.V1TestHelpersTestClocks.Retrieve(ctx, testClockID, nil)
		if err != nil {
			return nil, err
		}
		if clock.Status == stripe.TestHelpersTestClockStatusReady {
			return clock, nil
		}
		time.Sleep(1 * time.Second)
	}
	return nil, fmt.Errorf("test clock %s did not become ready", testClockID)
}

func syncSubscriptionNowToTestClock(ctx context.Context, sc *stripe.Client, testClockID string) {
	clock, err := sc.V1TestHelpersTestClocks.Retrieve(ctx, testClockID, nil)
	if err != nil {
		panic(err)
	}
	website.SetSubscriptionNowForTests(time.Unix(clock.FrozenTime, 0))
}

func deleteTestClock(ctx context.Context, sc *stripe.Client, testClockID string) {
	_, err := sc.V1TestHelpersTestClocks.Delete(ctx, testClockID, nil)
	if err != nil {
		fmt.Printf("      warning: failed to delete test clock %s: %v\n", testClockID, err)
	}
}

func validateStoredSubscriptionData(ctx context.Context, pool *pgxpool.Pool, userID int, customerID string, subscriptionID string) error {
	user, err := db.QueryOne[models.User](ctx, pool, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
	if err != nil {
		return err
	}
	if user.StripeCustomerID == nil || *user.StripeCustomerID != customerID {
		return fmt.Errorf("stored stripe_customer_id mismatch")
	}
	if user.StripeSubscriptionID == nil || *user.StripeSubscriptionID != subscriptionID {
		return fmt.Errorf("stored stripe_subscription_id mismatch")
	}
	if user.SubscriptionStatus == nil || *user.SubscriptionStatus == "" {
		return fmt.Errorf("stored subscription_status is empty")
	}
	return nil
}

func createSubscriptionTestUser(ctx context.Context, pool *pgxpool.Pool, username string) (int, string) {
	emailAddress := uuid.New().String() + "@example.com"
	hashedPassword := auth.HashPassword("password")

	var userID int
	err := pool.QueryRow(ctx, `
		INSERT INTO hmn_user (username, email, password, date_joined, registration_ip, status)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, username, emailAddress, hashedPassword.String(), time.Now(), net.ParseIP("127.0.0.1"), models.UserStatusConfirmed).Scan(&userID)
	if err != nil {
		panic(err)
	}

	return userID, emailAddress
}

func getSubscriptionPeriodEndFromStripe(sub *stripe.Subscription) *time.Time {
	if sub == nil || sub.Items == nil || len(sub.Items.Data) == 0 {
		return nil
	}

	t := time.Unix(sub.Items.Data[0].CurrentPeriodEnd, 0)
	return &t
}

func printSubscriptionData(ctx context.Context, pool *pgxpool.Pool, userID int) {
	user, err := db.QueryOne[models.User](ctx, pool, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nStored user membership data:\n")
	fmt.Printf("  user_id: %d\n", user.ID)
	fmt.Printf("  username: %s\n", user.Username)
	fmt.Printf("  is_subscribed: %v\n", user.IsSubscribed)
	fmt.Printf("  stripe_customer_id: %s\n", stringOrEmpty(user.StripeCustomerID))
	fmt.Printf("  stripe_subscription_id: %s\n", stringOrEmpty(user.StripeSubscriptionID))
	fmt.Printf("  subscription_status: %s\n", stringOrEmpty(user.SubscriptionStatus))
	if user.CurrentPeriodEnd != nil {
		fmt.Printf("  current_period_end: %s\n", user.CurrentPeriodEnd.UTC().Format(time.RFC3339))
	} else {
		fmt.Printf("  current_period_end: \n")
	}
	fmt.Printf("  cancel_at_period_end: %v\n", user.CancelAtPeriodEnd)
	if user.GracePeriodStartedAt != nil {
		fmt.Printf("  grace_period_started_at: %s\n", user.GracePeriodStartedAt.UTC().Format(time.RFC3339))
	}
	if user.GracePeriodEndsAt != nil {
		fmt.Printf("  grace_period_ends_at: %s\n", user.GracePeriodEndsAt.UTC().Format(time.RFC3339))
	}
	fmt.Printf("  grace_available: %v\n", user.GraceAvailable)

	payments, err := db.Query[models.UserPayment](ctx, pool, `
		SELECT $columns
		FROM user_payment
		WHERE user_id = $1
		ORDER BY paid_at DESC
	`, userID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nStored payment rows: %d\n", len(payments))
	for i, payment := range payments {
		fmt.Printf("  [%d] invoice=%s amount_cents=%d currency=%s paid_at=%s\n",
			i,
			stringOrEmpty(payment.StripeInvoiceID),
			payment.AmountCents,
			payment.Currency,
			payment.PaidAt.UTC().Format(time.RFC3339),
		)
	}
}

func stringOrEmpty(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func isExpectedACHVerificationPending(err error) bool {
	var stripeErr *stripe.Error
	if errors.As(err, &stripeErr) {
		return strings.Contains(stripeErr.Msg, "must be verified before they can be attached to a customer")
	}
	return strings.Contains(err.Error(), "must be verified before they can be attached to a customer")
}

func isStripeCardDeclined(err error) bool {
	if err == nil {
		return false
	}
	if strings.Contains(err.Error(), "card_declined") {
		return true
	}
	var stripeErr *stripe.Error
	if errors.As(err, &stripeErr) {
		return stripeErr.Code == stripe.ErrorCodeCardDeclined ||
			stripeErr.Type == stripe.ErrorTypeCard ||
			stripeErr.DeclineCode == stripe.DeclineCodeGenericDecline
	}
	return false
}

func persistPendingVerificationState(ctx context.Context, pool *pgxpool.Pool, userID int, customerID string) error {
	_, err := pool.Exec(ctx, `
		UPDATE hmn_user
		SET
			is_subscribed = false,
			stripe_customer_id = $1,
			subscription_status = 'pending_verification',
			current_period_end = NULL,
			cancel_at_period_end = false
		WHERE id = $2
	`, customerID, userID)
	return err
}

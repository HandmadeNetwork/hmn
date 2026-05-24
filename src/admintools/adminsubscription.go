package admintools

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/google/uuid"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-go/v84"
)

func addSubscriptionCommands(adminCommand *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "subscription",
		Short: "Admin commands for subscription testing",
	}
	adminCommand.AddCommand(cmd)

	addSubscriptionTestCommand(cmd)
}

func addSubscriptionTestCommand(subscriptionCommand *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run subscription test scenarios and print stored DB results",
		Run: func(cmd *cobra.Command, _ []string) {
			if config.Config.Stripe.SecretKey == "" || config.Config.Stripe.PriceID == "" {
				fmt.Fprintf(os.Stderr, "Stripe.SecretKey and Stripe.PriceID must be set in config.\n")
				os.Exit(1)
			}

			ctx := context.Background()
			conn := db.NewConn()
			defer conn.Close(ctx)

			sc := stripe.NewClient(config.Config.Stripe.SecretKey)
			scenarios := []subscriptionTestScenario{
				{
					Name: "Credit card (tok_visa)",
					CreatePaymentMethod: func(ctx context.Context, sc *stripe.Client) (*stripe.PaymentMethod, error) {
						return sc.V1PaymentMethods.Create(ctx, &stripe.PaymentMethodCreateParams{
							Type: stripe.String("card"),
							Card: &stripe.PaymentMethodCreateCardParams{
								Token: stripe.String("tok_visa"),
							},
						})
					},
				},
				{
					Name: "ACH (US bank account)",
					CreatePaymentMethod: func(ctx context.Context, sc *stripe.Client) (*stripe.PaymentMethod, error) {
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
					},
				},
			}

			failed := false
			for i, scenario := range scenarios {
				fmt.Printf("\n========== Scenario %d/%d: %s ==========\n", i+1, len(scenarios), scenario.Name)
				result, err := runSubscriptionScenario(ctx, conn, sc, scenario)
				if err != nil {
					failed = true
					fmt.Printf("RESULT: FAIL\n")
					fmt.Printf("ERROR: %v\n", err)
				} else if result == subscriptionTestResultPending {
					fmt.Printf("RESULT: PENDING (expected for ACH verification)\n")
				} else {
					fmt.Printf("RESULT: PASS\n")
				}
			}

			if failed {
				os.Exit(1)
			}
		},
	}

	subscriptionCommand.AddCommand(cmd)
}

type subscriptionTestScenario struct {
	Name                string
	CreatePaymentMethod func(context.Context, *stripe.Client) (*stripe.PaymentMethod, error)
}

type subscriptionTestResult int

const (
	subscriptionTestResultPass subscriptionTestResult = iota
	subscriptionTestResultPending
)

func runSubscriptionScenario(ctx context.Context, conn db.ConnOrTx, sc *stripe.Client, scenario subscriptionTestScenario) (subscriptionTestResult, error) {
	username := fmt.Sprintf("subtest_%s", uuid.NewString()[:8])
	fmt.Printf("[1/6] Creating test user: %s\n", username)
	userID, emailAddress := createSubscriptionTestUser(ctx, conn, username)
	fmt.Printf("      user_id=%d email=%s\n", userID, emailAddress)

	fmt.Printf("[2/6] Creating Stripe customer\n")
	customer, err := sc.V1Customers.Create(ctx, &stripe.CustomerCreateParams{
		Email: stripe.String(emailAddress),
		Name:  stripe.String(username),
		Metadata: map[string]string{
			"user_id": strconv.Itoa(userID),
		},
	})
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      customer_id=%s\n", customer.ID)

	fmt.Printf("[3/6] Creating payment method (%s)\n", scenario.Name)
	paymentMethod, err := scenario.CreatePaymentMethod(ctx, sc)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      payment_method_id=%s\n", paymentMethod.ID)

	fmt.Printf("[4/6] Attaching payment method and creating subscription\n")
	_, err = sc.V1PaymentMethods.Attach(ctx, paymentMethod.ID, &stripe.PaymentMethodAttachParams{
		Customer: stripe.String(customer.ID),
	})
	if err != nil {
		if isExpectedACHVerificationPending(err) {
			fmt.Printf("      ACH verification is pending; subscription will complete after verification.\n")
			if updateErr := persistPendingVerificationState(ctx, conn, userID, customer.ID); updateErr != nil {
				return subscriptionTestResultPass, updateErr
			}
			printSubscriptionData(ctx, conn, userID)
			return subscriptionTestResultPending, nil
		}
		return subscriptionTestResultPass, err
	}

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
	subscriptionParams.AddExpand("latest_invoice")

	subscription, err := sc.V1Subscriptions.Create(ctx, subscriptionParams)
	if err != nil {
		return subscriptionTestResultPass, err
	}
	fmt.Printf("      subscription_id=%s status=%s\n", subscription.ID, subscription.Status)

	fmt.Printf("[5/6] Writing subscription state to database\n")
	renewalDate := getSubscriptionPeriodEndFromStripe(subscription)
	isSubscribed := subscription.Status == stripe.SubscriptionStatusActive || subscription.Status == stripe.SubscriptionStatusTrialing
	_, err = conn.Exec(ctx, `
		UPDATE hmn_user
		SET
			is_subscribed = $1,
			stripe_customer_id = $2,
			stripe_subscription_id = $3,
			subscription_status = $4,
			current_period_end = $5,
			cancel_at_period_end = $6
		WHERE id = $7
	`, isSubscribed, customer.ID, subscription.ID, subscription.Status, renewalDate, subscription.CancelAtPeriodEnd, userID)
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
		_, err = conn.Exec(ctx, `
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

	fmt.Printf("[6/6] Verifying and printing stored subscription data\n")
	if err := validateStoredSubscriptionData(ctx, conn, userID, customer.ID, subscription.ID); err != nil {
		return subscriptionTestResultPass, err
	}
	printSubscriptionData(ctx, conn, userID)
	return subscriptionTestResultPass, nil
}

func validateStoredSubscriptionData(ctx context.Context, conn db.ConnOrTx, userID int, customerID string, subscriptionID string) error {
	user, err := db.QueryOne[models.User](ctx, conn, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
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

func createSubscriptionTestUser(ctx context.Context, conn db.ConnOrTx, username string) (int, string) {
	emailAddress := uuid.New().String() + "@example.com"
	hashedPassword := auth.HashPassword("password")

	var userID int
	err := conn.QueryRow(ctx, `
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

func printSubscriptionData(ctx context.Context, conn db.ConnOrTx, userID int) {
	user, err := db.QueryOne[models.User](ctx, conn, "SELECT $columns FROM hmn_user WHERE id = $1", userID)
	if err != nil {
		panic(err)
	}

	fmt.Printf("\nStored user subscription data:\n")
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

	payments, err := db.Query[models.UserPayment](ctx, conn, `
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

func persistPendingVerificationState(ctx context.Context, conn db.ConnOrTx, userID int, customerID string) error {
	_, err := conn.Exec(ctx, `
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

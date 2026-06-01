package admintools

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stripe/stripe-go/v84"
)

type subscriptionTestScenario struct {
	Name                string
	CreatePaymentMethod func(context.Context, *stripe.Client) (*stripe.PaymentMethod, error)
	Run                 func(context.Context, *pgxpool.Pool, *stripe.Client) (subscriptionTestResult, error)
}

type subscriptionTestResult int

const (
	subscriptionTestResultPass subscriptionTestResult = iota
	subscriptionTestResultPending
)

func membershipScenarios() []subscriptionTestScenario {
	return []subscriptionTestScenario{
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
			Name: "Credit card one-time charge (EUR)",
			Run:  runEuroCardChargeScenario,
		},
		{
			Name: "Credit card declined (tok_chargeDeclined)",
			Run:  runDeclinedCardScenario,
		},
		{
			Name: "ACH (US bank account)",
			CreatePaymentMethod: createACHPaymentMethod,
		},
		{
			Name: "ACH grace expires after 2 week clock advance",
			Run:  runACHGraceExpiryScenario,
		},
		{
			Name: "ACH verification after 2 day clock advance",
			Run:  runACHVerificationAfterAdvanceScenario,
		},
		{
			Name: "Card renewal failure → grace → payment method update",
			Run:  runCardRenewalFailureGraceRecoveryScenario,
		},
	}
}

func runSubscriptionScenario(ctx context.Context, pool *pgxpool.Pool, sc *stripe.Client, scenario subscriptionTestScenario) (subscriptionTestResult, error) {
	if scenario.Run != nil {
		return scenario.Run(ctx, pool, sc)
	}
	return runCardOrACHScenario(ctx, pool, sc, scenario)
}

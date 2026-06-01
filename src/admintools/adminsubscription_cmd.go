package admintools

import (
	"context"
	"fmt"
	"os"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"github.com/spf13/cobra"
	"github.com/stripe/stripe-go/v84"
)

func addSubscriptionCommands(adminCommand *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "membership",
		Short: "Admin commands for membership testing",
	}
	adminCommand.AddCommand(cmd)

	legacyCmd := &cobra.Command{
		Use:    "subscription",
		Short:  "Alias for membership commands",
		Hidden: true,
	}
	adminCommand.AddCommand(legacyCmd)

	addSubscriptionTestCommand(cmd)
	addSubscriptionTestCommand(legacyCmd)
}

func addSubscriptionTestCommand(subscriptionCommand *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "Run membership test scenarios and print stored DB results",
		Run: func(cmd *cobra.Command, _ []string) {
			if config.Config.Stripe.SecretKey == "" || config.Config.Stripe.PriceID == "" {
				fmt.Fprintf(os.Stderr, "Stripe.SecretKey and Stripe.PriceID must be set in config.\n")
				os.Exit(1)
			}

			ctx := context.Background()
			pool := db.NewConnPool()
			defer pool.Close()

			if override := config.Config.Stripe.SubscriptionNowOverride; override != "" {
				fmt.Printf("Using membership time override: %s\n", override)
			}
			if testClockID := config.Config.Stripe.TestClockID; testClockID != "" {
				fmt.Printf("Using Stripe test clock: %s\n", testClockID)
			}

			sc := stripe.NewClient(config.Config.Stripe.SecretKey)
			scenarios := membershipScenarios()

			failed := false
			for i, scenario := range scenarios {
				fmt.Printf("\n========== Scenario %d/%d: %s ==========\n", i+1, len(scenarios), scenario.Name)
				result, err := runSubscriptionScenario(ctx, pool, sc, scenario)
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

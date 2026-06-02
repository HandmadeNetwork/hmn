package admintools

import (
	"context"
	"errors"
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
	addSubscriptionInspectCommand(cmd)
	addSubscriptionInspectCommand(legacyCmd)
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
			passCount := 0
			pendingCount := 0
			failCount := 0
			var failedScenarioNames []string
			for i, scenario := range scenarios {
				fmt.Printf("\n========== Scenario %d/%d: %s ==========\n", i+1, len(scenarios), scenario.Name)
				result, err := runSubscriptionScenario(ctx, pool, sc, scenario)
				if err != nil {
					failed = true
					failCount++
					failedScenarioNames = append(failedScenarioNames, scenario.Name)
					fmt.Printf("RESULT: FAIL\n")
					fmt.Printf("ERROR: %v\n", err)
				} else if result == subscriptionTestResultPending {
					pendingCount++
					fmt.Printf("RESULT: PENDING (expected for ACH verification)\n")
				} else {
					passCount++
					fmt.Printf("RESULT: PASS\n")
				}
			}

			fmt.Printf("\n========== Membership Test Summary ==========\n")
			fmt.Printf("Total scenarios: %d\n", len(scenarios))
			fmt.Printf("PASS: %d\n", passCount)
			fmt.Printf("PENDING: %d\n", pendingCount)
			fmt.Printf("FAIL: %d\n", failCount)
			if len(failedScenarioNames) > 0 {
				fmt.Printf("Failed scenarios:\n")
				for _, name := range failedScenarioNames {
					fmt.Printf("  - %s\n", name)
				}
			}

			if failed {
				os.Exit(1)
			}
		},
	}

	subscriptionCommand.AddCommand(cmd)
}

func addSubscriptionInspectCommand(subscriptionCommand *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "inspect <username>",
		Short: "Print membership/payment debug info for a user",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			username := args[0]

			ctx := context.Background()
			pool := db.NewConnPool()
			defer pool.Close()

			userID, err := db.QueryOneScalar[int](ctx, pool, `
				SELECT id
				FROM hmn_user
				WHERE LOWER(username) = LOWER($1)
			`, username)
			if err != nil {
				if errors.Is(err, db.NotFound) {
					fmt.Printf("User not found: %s\n", username)
					os.Exit(1)
				}
				panic(err)
			}

			printSubscriptionData(ctx, pool, userID)
		},
	}

	subscriptionCommand.AddCommand(cmd)
}

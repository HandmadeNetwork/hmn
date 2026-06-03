package admintools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

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
	var scenarioFilter string
	var openMailpit bool

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

			originalEmailConfig := config.Config.Email
			defer func() {
				config.Config.Email = originalEmailConfig
			}()

			mailpit, mailpitInstalled, err := startMembershipMailpit()
			if err != nil {
				fmt.Printf("WARNING: failed to start Mailpit, email checks disabled: %v\n", err)
			}
			if !mailpitInstalled {
				fmt.Printf("Mailpit binary not found; skipping email checks.\n")
			}
			if mailpit != nil {
				fmt.Printf("Mailpit started: HTTP=%s SMTP=%s\n", mailpit.httpBaseURL, mailpit.smtpAddr)
				if openMailpit {
					if err := openURLInBrowser(mailpit.httpBaseURL); err != nil {
						fmt.Printf("WARNING: failed to open Mailpit UI: %v\n", err)
					}
				}
				defer func() {
					if stopErr := mailpit.Stop(); stopErr != nil {
						fmt.Printf("WARNING: failed to stop Mailpit: %v\n", stopErr)
					}
				}()
				ctx = withMembershipMailpit(ctx, mailpit)
			}

			if override := config.Config.Stripe.SubscriptionNowOverride; override != "" {
				fmt.Printf("Using membership time override: %s\n", override)
			}
			if testClockID := config.Config.Stripe.TestClockID; testClockID != "" {
				fmt.Printf("Using Stripe test clock: %s\n", testClockID)
			}

			sc := stripe.NewClient(config.Config.Stripe.SecretKey)
			scenarios := membershipScenarios()
			if scenarioFilter != "" {
				selected, err := selectMembershipScenarios(scenarios, scenarioFilter)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Invalid --scenario value %q: %v\n", scenarioFilter, err)
					os.Exit(1)
				}
				scenarios = selected
			}

			failed := false
			passCount := 0
			pendingCount := 0
			failCount := 0
			var failedScenarioNames []string
			for i, scenario := range scenarios {
				if mailpit != nil {
					if err := mailpit.ClearMessages(); err != nil {
						fmt.Printf("WARNING: failed to clear Mailpit mailbox before scenario: %v\n", err)
					}
				}

				fmt.Printf("\n========== Scenario %d/%d: %s ==========\n", i+1, len(scenarios), scenario.Name)
				result, err := runSubscriptionScenario(ctx, pool, sc, scenario)

				if mailpit != nil {
					subjects, subjErr := mailpit.messageSubjects()
					if subjErr != nil {
						fmt.Printf("EMAILS: unable to list received messages (%v)\n", subjErr)
					} else if len(subjects) == 0 {
						fmt.Printf("EMAILS: none received\n")
					} else {
						fmt.Printf("EMAILS: received %d\n", len(subjects))
						for _, subject := range subjects {
							fmt.Printf("  - %s\n", subject)
						}
					}
				}

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
	cmd.Flags().StringVar(&scenarioFilter, "scenario", "", "Run a single scenario by 1-based index or exact name")
	cmd.Flags().BoolVar(&openMailpit, "open-mailpit", false, "Open Mailpit web UI in the default browser when available")

	subscriptionCommand.AddCommand(cmd)
}

func selectMembershipScenarios(scenarios []subscriptionTestScenario, filter string) ([]subscriptionTestScenario, error) {
	if idx, err := strconv.Atoi(filter); err == nil {
		if idx < 1 || idx > len(scenarios) {
			return nil, fmt.Errorf("index out of range (1-%d)", len(scenarios))
		}
		return []subscriptionTestScenario{scenarios[idx-1]}, nil
	}

	needle := strings.TrimSpace(filter)
	if needle == "" {
		return nil, errors.New("scenario name is blank")
	}
	for _, scenario := range scenarios {
		if strings.EqualFold(scenario.Name, needle) {
			return []subscriptionTestScenario{scenario}, nil
		}
	}

	return nil, fmt.Errorf("not found; use 1-%d or one of the scenario names", len(scenarios))
}

func openURLInBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
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

package admintools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/jackc/pgx/v4"
	"github.com/spf13/cobra"
)

func init() {
	setPasswordCommand := &cobra.Command{
		Use:   "setpassword [username] [new password]",
		Short: "Replace a user's password",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Printf("You must provide a username and a password.\n\n")
				cmd.Usage()
				os.Exit(1)
			}

			username := args[0]
			password := args[1]

			ctx := context.Background()
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			row := conn.QueryRow(ctx, "SELECT id, username FROM auth_user WHERE lower(username) = lower($1)", username)
			var id int
			var canonicalUsername string
			err := row.Scan(&id, &canonicalUsername)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					fmt.Printf("User '%s' not found\n", username)
					os.Exit(1)
				} else {
					panic(err)
				}
			}

			hashedPassword := auth.HashPassword(password)

			err = auth.UpdatePassword(ctx, conn, canonicalUsername, hashedPassword)
			if err != nil {
				panic(err)
			}

			fmt.Printf("Successfully updated password for '%s'\n", canonicalUsername)
		},
	}
	website.WebsiteCommand.AddCommand(setPasswordCommand)

	activateUserCommand := &cobra.Command{
		Use:   "activateuser [username]",
		Short: "Activates a user manually",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 1 {
				fmt.Printf("You must provide a username.\n\n")
				cmd.Usage()
				os.Exit(1)
			}

			username := args[0]

			ctx := context.Background()
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			res, err := conn.Exec(ctx, "UPDATE auth_user SET status = $1 WHERE LOWER(username) = LOWER($2);", models.UserStatusActive, username)
			if err != nil {
				panic(err)
			}
			if res.RowsAffected() == 0 {
				fmt.Printf("User not found.\n\n")
			}

			fmt.Printf("User has been successfully activated.\n\n")
		},
	}
	website.WebsiteCommand.AddCommand(activateUserCommand)

	sendTestMailCommand := &cobra.Command{
		Use:   "sendtestmail [type] [toAddress] [toName]",
		Short: "Sends a test mail",
		Run: func(cmd *cobra.Command, args []string) {
			templates.Init()
			if len(args) < 3 {
				fmt.Printf("You must provide the email type and recipient details.\n\n")
				cmd.Usage()
				os.Exit(1)
			}

			emailType := args[0]
			toAddress := args[1]
			toName := args[2]

			p := perf.MakeNewRequestPerf("admintools", "email test", emailType)
			var err error
			switch emailType {
			case "registration":
				err = email.SendRegistrationEmail(toAddress, toName, "test_user", "test_token", p)
			case "passwordreset":
				err = email.SendPasswordReset(toAddress, toName, "test_user", "test_token", time.Now().Add(time.Hour*24), p)
			default:
				fmt.Printf("You must provide a valid email type\n\n")
				cmd.Usage()
				os.Exit(1)
			}
			p.EndRequest()
			perf.LogPerf(p, logging.Info())
			if err != nil {
				panic(oops.New(err, "Failed to send test email"))
			}
		},
	}
	website.WebsiteCommand.AddCommand(sendTestMailCommand)
}

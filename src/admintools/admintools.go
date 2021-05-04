package admintools

import (
	"context"
	"errors"
	"fmt"
	"os"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/db"
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

			hashedPassword, err := auth.HashPassword(password)
			if err != nil {
				panic(err)
			}

			err = auth.UpdatePassword(ctx, conn, canonicalUsername, hashedPassword)
			if err != nil {
				panic(err)
			}

			fmt.Printf("Successfully updated password for '%s'\n", canonicalUsername)
		},
	}

	website.WebsiteCommand.AddCommand(setPasswordCommand)
}

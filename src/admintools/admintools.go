package admintools

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strconv"
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
	adminCommand := &cobra.Command{
		Use:   "admin",
		Short: "Miscellaneous admin commands",
	}
	website.WebsiteCommand.AddCommand(adminCommand)

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
	adminCommand.AddCommand(setPasswordCommand)

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

			res, err := conn.Exec(ctx, "UPDATE auth_user SET status = $1 WHERE LOWER(username) = LOWER($2);", models.UserStatusConfirmed, username)
			if err != nil {
				panic(err)
			}
			if res.RowsAffected() == 0 {
				fmt.Printf("User not found.\n\n")
			}

			fmt.Printf("User has been successfully activated.\n\n")
		},
	}
	adminCommand.AddCommand(activateUserCommand)

	userStatusCommand := &cobra.Command{
		Use:   "userstatus [username] [status]",
		Short: "Set a user's status manually",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				fmt.Printf("You must provide a username and status.\n\n")
				fmt.Printf("Statuses:\n")
				fmt.Printf("1. inactive:\n")
				fmt.Printf("2. confirmed:\n")
				fmt.Printf("3. approved:\n")
				fmt.Printf("4. banned:\n")
				cmd.Usage()
				os.Exit(1)
			}

			username := args[0]
			statusStr := args[1]
			status := models.UserStatusInactive
			switch statusStr {
			case "inactive":
				status = models.UserStatusInactive
			case "confirmed":
				status = models.UserStatusConfirmed
			case "approved":
				status = models.UserStatusApproved
			case "banned":
				status = models.UserStatusBanned
			default:
				fmt.Printf("You must provide a valid status\n\n")
				fmt.Printf("Statuses:\n")
				fmt.Printf("1. inactive:\n")
				fmt.Printf("2. confirmed:\n")
				fmt.Printf("3. approved:\n")
				fmt.Printf("4. banned:\n")
				cmd.Usage()
				os.Exit(1)
			}

			ctx := context.Background()
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			res, err := conn.Exec(ctx, "UPDATE auth_user SET status = $1 WHERE LOWER(username) = LOWER($2);", status, username)
			if err != nil {
				panic(err)
			}
			if res.RowsAffected() == 0 {
				fmt.Printf("User not found.\n\n")
			}

			fmt.Printf("%s is now %s\n\n", username, statusStr)
		},
	}
	adminCommand.AddCommand(userStatusCommand)

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
	adminCommand.AddCommand(sendTestMailCommand)

	createSubforumCommand := &cobra.Command{
		Use:   "createsubforum",
		Short: "Create a new subforum",
		Run: func(cmd *cobra.Command, args []string) {
			name, _ := cmd.Flags().GetString("name")
			slug, _ := cmd.Flags().GetString("slug")
			blurb, _ := cmd.Flags().GetString("blurb")
			parentSlug, _ := cmd.Flags().GetString("parent_slug")
			projectSlug, _ := cmd.Flags().GetString("project_slug")

			ctx := context.Background()
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			tx, err := conn.Begin(ctx)
			if err != nil {
				panic(err)
			}
			defer tx.Rollback(ctx)

			projectId, err := db.QueryInt(ctx, tx, `SELECT id FROM handmade_project WHERE slug = $1`, projectSlug)
			if err != nil {
				panic(err)
			}

			var parentId *int
			if parentSlug == "" {
				// Select the root subforum
				id, err := db.QueryInt(ctx, tx,
					`SELECT id FROM handmade_subforum WHERE parent_id IS NULL AND project_id = $1`,
					projectId,
				)
				if err != nil {
					panic(err)
				}
				parentId = &id
			} else {
				// Select the parent
				id, err := db.QueryInt(ctx, tx,
					`SELECT id FROM handmade_subforum WHERE slug = $1 AND project_id = $2`,
					parentSlug, projectId,
				)
				if err != nil {
					panic(err)
				}
				parentId = &id
			}

			newId, err := db.QueryInt(ctx, tx,
				`
				INSERT INTO handmade_subforum (name, slug, blurb, parent_id, project_id)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING id
				`,
				name,
				slug,
				blurb,
				parentId,
				projectId,
			)
			if err != nil {
				panic(err)
			}

			err = tx.Commit(ctx)
			if err != nil {
				panic(err)
			}

			fmt.Printf("Created new subforum with id: %d\n", newId)
		},
	}
	createSubforumCommand.Flags().String("name", "", "")
	createSubforumCommand.Flags().String("slug", "", "")
	createSubforumCommand.Flags().String("blurb", "", "")
	createSubforumCommand.Flags().String("parent_slug", "", "")
	createSubforumCommand.Flags().String("project_slug", "", "")
	createSubforumCommand.MarkFlagRequired("name")
	createSubforumCommand.MarkFlagRequired("slug")
	createSubforumCommand.MarkFlagRequired("project_slug")
	adminCommand.AddCommand(createSubforumCommand)

	moveThreadsToSubforumCommand := &cobra.Command{
		Use:   "movethreadstosubforum [<thread id>...]",
		Short: "Move threads to a subforum, changing their type if necessary",
		Run: func(cmd *cobra.Command, args []string) {
			projectSlug, _ := cmd.Flags().GetString("project_slug")
			subforumSlug, _ := cmd.Flags().GetString("subforum_slug")

			ctx := context.Background()
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			tx, err := conn.Begin(ctx)
			if err != nil {
				panic(err)
			}
			defer tx.Rollback(ctx)

			projectId, err := db.QueryInt(ctx, tx, `SELECT id FROM handmade_project WHERE slug = $1`, projectSlug)
			if err != nil {
				panic(err)
			}

			subforumId, err := db.QueryInt(ctx, tx,
				`SELECT id FROM handmade_subforum WHERE slug = $1 AND project_id = $2`,
				subforumSlug, projectId,
			)
			if err != nil {
				panic(err)
			}

			var threadIds []int
			for _, threadIdStr := range args {
				threadId, err := strconv.Atoi(threadIdStr)
				if err != nil {
					fmt.Printf("Couldn't move thread '%s': couldn't parse ID\n", threadIdStr)
					continue
				}
				threadIds = append(threadIds, threadId)
			}

			threadsTag, err := tx.Exec(ctx,
				`
				UPDATE handmade_thread
				SET
					project_id = $2,
					subforum_id = $3,
					personal_article_user_id = NULL,
					type = 2
				WHERE
					id = ANY ($1)
				`,
				threadIds,
				projectId,
				subforumId,
			)

			postsTag, err := tx.Exec(ctx,
				`
				UPDATE handmade_post
				SET
					thread_type = 2
				WHERE
					thread_id = ANY ($1)
				`,
				threadIds,
			)

			err = tx.Commit(ctx)
			if err != nil {
				panic(err)
			}

			fmt.Printf("Successfully moved %d threads (and %d posts).\n", threadsTag.RowsAffected(), postsTag.RowsAffected())
		},
	}
	moveThreadsToSubforumCommand.Flags().String("project_slug", "", "")
	moveThreadsToSubforumCommand.Flags().String("subforum_slug", "", "")
	moveThreadsToSubforumCommand.MarkFlagRequired("project_slug")
	moveThreadsToSubforumCommand.MarkFlagRequired("subforum_slug")
	adminCommand.AddCommand(moveThreadsToSubforumCommand)

	addProjectCommands(adminCommand)
}

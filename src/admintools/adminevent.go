package admintools

import (
	"context"
	"fmt"
	"os"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/spf13/cobra"
)

func addEventCommands(adminCommand *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "event",
		Short: "Admin commands for managing events",
	}
	adminCommand.AddCommand(cmd)

	addJamNagCommand(cmd)
}

func addJamNagCommand(postCommand *cobra.Command) {
	cmd := &cobra.Command{
		Use:   "jam-nag <slug>",
		Short: "Manually nag users to finish signing up for an upcoming jam",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			conn := db.NewConn()
			defer conn.Close(ctx)

			if len(args) < 1 {
				cmd.Usage()
				os.Exit(1)
			}
			jam, ok := hmndata.JamBySlug(args[0])
			if !ok {
				fmt.Fprintf(os.Stderr, "No jam found with slug %s.\n", args[0])
				fmt.Fprintf(os.Stderr, "Upcoming options:\n")
				for _, jam := range hmndata.AllJams {
					if time.Now().Before(jam.StartTime) {
						fmt.Fprintf(os.Stderr, "- %s\n", jam.Slug)
					}
				}
				os.Exit(1)
			}

			nags, err := website.NagUsersToCreateJamProjects(ctx, conn, &jam)
			if err != nil {
				fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
			} else {
				fmt.Printf("Users nagged:\n")
				for _, nag := range nags {
					fmt.Printf("- %s\n", nag)
				}
			}
		},
	}

	postCommand.AddCommand(cmd)
}

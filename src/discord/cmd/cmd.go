package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/spf13/cobra"
)

func init() {
	rootCommand := &cobra.Command{
		Use:   "discord",
		Short: "Commands for interacting with Discord",
	}
	website.WebsiteCommand.AddCommand(rootCommand)

	scrapeCommand := &cobra.Command{
		Use:   "scrapechannel [<channel id>...]",
		Short: "Scrape the entire history of Discord channels",
		Long:  "Scrape the entire history of Discord channels, saving message content (but not creating snippets)",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			conn := db.NewConnPool()
			defer conn.Close()

			for _, channelID := range args {
				discord.ScrapeAll(ctx, conn, channelID, time.Time{})
			}
		},
	}
	rootCommand.AddCommand(scrapeCommand)

	makeSnippetCommand := &cobra.Command{
		Use:   "makesnippet <channel id> [<message id>...]",
		Short: "Make snippets from Discord messages",
		Long:  "Creates snippets from the specified messages in the specified channel. Will create a snippet as long as the poster of the message linked their account regardless of user settings.",
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) < 2 {
				cmd.Usage()
				os.Exit(1)
			}
			ctx := context.Background()
			conn := db.NewConn()
			defer conn.Close(ctx)

			chanID := args[0]

			count := 0

			for _, msgID := range args[1:] {
				message, err := discord.GetChannelMessage(ctx, chanID, msgID)
				if errors.Is(err, discord.NotFound) {
					logging.Warn().Msg(fmt.Sprintf("no message found on discord for id %s", msgID))
					continue
				} else if err != nil {
					logging.Error().Msg(fmt.Sprintf("failed to fetch discord message id %s", msgID))
					continue
				}
				err = discord.TrackMessage(ctx, conn, message)
				if err != nil {
					logging.Error().Msg(fmt.Sprintf("failed to track discord message id %s", msgID))
					continue
				}
				err = discord.UpdateInternedMessage(ctx, conn, message, false, true, false)
				if err != nil {
					logging.Error().Msg(fmt.Sprintf("failed to handle interned message id %s", msgID))
					continue
				}
				count += 1
			}

			logging.Info().Msg(fmt.Sprintf("Handled %d messages", count))
		},
	}
	rootCommand.AddCommand(makeSnippetCommand)

	syncSupporterRolesCommand := &cobra.Command{
		Use:   "sync-supporter-roles",
		Short: "Sync supporter Discord roles for subscribed members",
		Long:  "Grants SupporterRoleID to subscribed users with linked Discord accounts, and removes it from others.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			conn := db.NewConnPool()
			defer conn.Close()

			dryRun, _ := cmd.Flags().GetBool("dry-run")

			userIDPtrs, err := db.Query[int](ctx, conn, `
				SELECT hmn_user.id
				FROM hmn_user
				INNER JOIN discord_user ON discord_user.hmn_user_id = hmn_user.id
				WHERE hmn_user.is_subscribed = true
			`)
			if err != nil {
				logging.Error().Err(err).Msg("failed to list subscribed users with Discord")
				os.Exit(1)
			}

			if dryRun {
				logging.Info().Int("count", len(userIDPtrs)).Msg("dry run: would sync supporter Discord roles")
				return
			}

			for _, userID := range userIDPtrs {
				website.SyncSupporterDiscordRole(ctx, conn, *userID)
			}
			logging.Info().Int("count", len(userIDPtrs)).Msg("synced supporter Discord roles")
		},
	}
	syncSupporterRolesCommand.Flags().Bool("dry-run", false, "log how many users would be synced without calling Discord")
	rootCommand.AddCommand(syncSupporterRolesCommand)
}

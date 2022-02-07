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
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			for _, channelID := range args {
				discord.Scrape(ctx, conn, channelID, time.Time{}, false)
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
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

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
				err = discord.InternMessage(ctx, conn, message)
				if err != nil {
					logging.Error().Msg(fmt.Sprintf("failed to intern discord message id %s", msgID))
					continue
				}
				err = discord.HandleInternedMessage(ctx, conn, message, false, true)
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
}

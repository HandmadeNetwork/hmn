package cmd

import (
	"context"
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
		Use:   "makesnippet [<message id>...]",
		Short: "Make snippets from saved Discord messages",
		Long:  "Make snippets from Discord messages whose content we have already saved. Useful for creating snippets from messages in non-showcase channels.",
		Run: func(cmd *cobra.Command, args []string) {
			ctx := context.Background()
			conn := db.NewConnPool(1, 1)
			defer conn.Close()

			err := discord.CreateMessageSnippets(ctx, conn, args...)
			if err != nil {
				logging.Error().Err(err).Msg("failed to create snippets")
			}
		},
	}
	rootCommand.AddCommand(makeSnippetCommand)
}

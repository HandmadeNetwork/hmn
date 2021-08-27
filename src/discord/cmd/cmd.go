package cmd

import (
	"context"
	"time"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/website"
	"github.com/spf13/cobra"
)

func init() {
	scrapeCommand := &cobra.Command{
		Use:   "discordscrapechannel [<channel id>...]",
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

	website.WebsiteCommand.AddCommand(scrapeCommand)
}

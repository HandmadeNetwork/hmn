package discord

import (
	"context"

	"git.handmade.network/hmn/hmn/src/oops"
)

func (bot *discordBotInstance) processLibraryMsg(ctx context.Context, msg *Message) error {
	switch msg.Type {
	case MessageTypeDefault, MessageTypeReply, MessageTypeApplicationCommand:
	default:
		return nil
	}

	if !msg.OriginalHasFields("content") {
		return nil
	}

	if !messageHasLinks(msg.Content) {
		err := DeleteMessage(ctx, msg.ChannelID, msg.ID)
		if err != nil {
			return oops.New(err, "failed to delete message")
		}

		if !msg.Author.IsBot {
			channel, err := CreateDM(ctx, msg.Author.ID)
			if err != nil {
				return oops.New(err, "failed to create DM channel")
			}

			err = SendMessages(ctx, bot.dbConn, MessageToSend{
				ChannelID: channel.ID,
				Req: CreateMessageRequest{
					Content: "Posts in #the-library are required to have a link. Discuss library content in other relevant channels.",
				},
			})
			if err != nil {
				return oops.New(err, "failed to send showcase warning message")
			}
		}
	}

	return nil
}

package discord

import (
	"context"
	"net/url"
	"regexp"
	"strings"

	"git.handmade.network/hmn/hmn/src/oops"
)

var reDiscordMessageLink = regexp.MustCompile(`https?://.+?(\s|$)`)

func (bot *discordBotInstance) processShowcaseMsg(ctx context.Context, msg *Message) error {
	switch msg.Type {
	case MessageTypeDefault, MessageTypeReply, MessageTypeApplicationCommand:
	default:
		return nil
	}

	hasGoodContent := true
	if originalMessageHasField(msg, "content") && !messageHasLinks(msg.Content) {
		hasGoodContent = false
	}

	hasGoodAttachments := true
	if originalMessageHasField(msg, "attachments") && len(msg.Attachments) == 0 {
		hasGoodAttachments = false
	}

	if !hasGoodContent && !hasGoodAttachments {
		err := DeleteMessage(ctx, msg.ChannelID, msg.ID)
		if err != nil {
			return oops.New(err, "failed to delete message")
		}

		if msg.Author != nil && !msg.Author.IsBot {
			channel, err := CreateDM(ctx, msg.Author.ID)
			if err != nil {
				return oops.New(err, "failed to create DM channel")
			}

			err = SendMessages(ctx, bot.dbConn, MessageToSend{
				ChannelID: channel.ID,
				Req: CreateMessageRequest{
					Content: "Posts in #project-showcase are required to have either an image/video or a link. Discuss showcase content in #projects.",
				},
			})
			if err != nil {
				return oops.New(err, "failed to send showcase warning message")
			}
		}
	}

	return nil
}

func (bot *discordBotInstance) processLibraryMsg(ctx context.Context, msg *Message) error {
	switch msg.Type {
	case MessageTypeDefault, MessageTypeReply, MessageTypeApplicationCommand:
	default:
		return nil
	}

	if !originalMessageHasField(msg, "content") {
		return nil
	}

	if !messageHasLinks(msg.Content) {
		err := DeleteMessage(ctx, msg.ChannelID, msg.ID)
		if err != nil {
			return oops.New(err, "failed to delete message")
		}

		if msg.Author != nil && !msg.Author.IsBot {
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

func messageHasLinks(content string) bool {
	links := reDiscordMessageLink.FindAllString(content, -1)
	for _, link := range links {
		_, err := url.Parse(strings.TrimSpace(link))
		if err == nil {
			return true
		}
	}

	return false
}

func originalMessageHasField(msg *Message, field string) bool {
	if msg.originalMap == nil {
		return false
	}

	_, ok := msg.originalMap[field]
	return ok
}

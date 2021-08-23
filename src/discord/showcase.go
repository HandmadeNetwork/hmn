package discord

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

var reDiscordMessageLink = regexp.MustCompile(`https?://.+?(\s|$)`)

var errNotEnoughInfo = errors.New("Discord didn't send enough info in this event for us to do this")

// TODO: Can this function be called asynchronously?
func (bot *botInstance) processShowcaseMsg(ctx context.Context, msg *Message) error {
	switch msg.Type {
	case MessageTypeDefault, MessageTypeReply, MessageTypeApplicationCommand:
	default:
		return nil
	}

	didDelete, err := bot.maybeDeleteShowcaseMsg(ctx, msg)
	if err != nil {
		return err
	}
	if didDelete {
		return nil
	}

	tx, err := bot.dbConn.Begin(ctx)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(ctx)

	// save the message, maybe save its contents, and maybe make a snippet too
	_, err = bot.saveMessageAndContents(ctx, tx, msg)
	if errors.Is(err, errNotEnoughInfo) {
		logging.ExtractLogger(ctx).Warn().
			Interface("msg", msg).
			Msg("didn't have enough info to process Discord message")
		return nil
	} else if err != nil {
		return err
	}
	if doSnippet, err := bot.allowedToCreateMessageSnippet(ctx, msg); doSnippet && err == nil {
		_, err := bot.createMessageSnippet(ctx, msg)
		if err != nil {
			return oops.New(err, "failed to create snippet in gateway")
		}
	} else if err != nil {
		return oops.New(err, "failed to check snippet permissions in gateway")
	}

	err = tx.Commit(ctx)
	if err != nil {
		return oops.New(err, "failed to commit Discord message updates")
	}

	return nil
}

func (bot *botInstance) maybeDeleteShowcaseMsg(ctx context.Context, msg *Message) (didDelete bool, err error) {
	hasGoodContent := true
	if msg.OriginalHasFields("content") && !messageHasLinks(msg.Content) {
		hasGoodContent = false
	}

	hasGoodAttachments := true
	if msg.OriginalHasFields("attachments") && len(msg.Attachments) == 0 {
		hasGoodAttachments = false
	}

	didDelete = false
	if !hasGoodContent && !hasGoodAttachments {
		didDelete = true
		err := DeleteMessage(ctx, msg.ChannelID, msg.ID)
		if err != nil {
			return false, oops.New(err, "failed to delete message")
		}

		if !msg.Author.IsBot {
			channel, err := CreateDM(ctx, msg.Author.ID)
			if err != nil {
				return false, oops.New(err, "failed to create DM channel")
			}

			err = SendMessages(ctx, bot.dbConn, MessageToSend{
				ChannelID: channel.ID,
				Req: CreateMessageRequest{
					Content: "Posts in #project-showcase are required to have either an image/video or a link. Discuss showcase content in #projects.",
				},
			})
			if err != nil {
				return false, oops.New(err, "failed to send showcase warning message")
			}
		}
	}

	return didDelete, nil
}

/*
Ensures that a Discord message is stored in the database. This function is
idempotent and can be called regardless of whether the item already exists in
the database.

This does not create snippets or do anything besides save the message itself.
*/
func (bot *botInstance) saveMessage(
	ctx context.Context,
	tx pgx.Tx,
	msg *Message,
) (*models.DiscordMessage, error) {
	iDiscordMessage, err := db.QueryOne(ctx, tx, models.DiscordMessage{},
		`
		SELECT $columns
		FROM handmade_discordmessage
		WHERE id = $1
		`,
		msg.ID,
	)
	if errors.Is(err, db.ErrNoMatchingRows) {
		if !msg.OriginalHasFields("author", "timestamp") {
			return nil, errNotEnoughInfo
		}

		_, err = tx.Exec(ctx,
			`
			INSERT INTO handmade_discordmessage (id, channel_id, guild_id, url, user_id, sent_at, snippet_created)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			`,
			msg.ID,
			msg.ChannelID,
			*msg.GuildID,
			msg.JumpURL(),
			msg.Author.ID,
			msg.Time(),
			false,
		)
		if err != nil {
			return nil, oops.New(err, "failed to save new discord message")
		}

		/*
			TODO(db): This is a spot where it would be really nice to be able
			to use RETURNING, and avoid this second query.
		*/
		iDiscordMessage, err = db.QueryOne(ctx, tx, models.DiscordMessage{},
			`
			SELECT $columns
			FROM handmade_discordmessage
			WHERE id = $1
			`,
			msg.ID,
		)
		if err != nil {
			panic(err)
		}
	} else if err != nil {
		return nil, oops.New(err, "failed to check for existing Discord message")
	}

	return iDiscordMessage.(*models.DiscordMessage), nil
}

/*
Processes a single Discord message, saving as much of the message's content
and attachments as allowed by our rules and user settings. Does NOT create
snippets.

Idempotent; can be called any time whether the message exists or not.
*/
func (bot *botInstance) saveMessageAndContents(
	ctx context.Context,
	tx pgx.Tx,
	msg *Message,
) (*models.DiscordMessage, error) {
	newMsg, err := bot.saveMessage(ctx, tx, msg)
	if err != nil {
		return nil, err
	}

	// Check for linked Discord user
	iDiscordUser, err := db.QueryOne(ctx, tx, models.DiscordUser{},
		`
		SELECT $columns
		FROM handmade_discorduser
		WHERE userid = $1
		`,
		msg.Author.ID,
	)
	if errors.Is(err, db.ErrNoMatchingRows) {
		return newMsg, nil
	} else if err != nil {
		return nil, oops.New(err, "failed to look up linked Discord user")
	}
	discordUser := iDiscordUser.(*models.DiscordUser)

	// We have a linked Discord account, so save the message contents (regardless of
	// whether we create a snippet or not).

	_, err = tx.Exec(ctx,
		`
		INSERT INTO handmade_discordmessagecontent (message_id, discord_id, last_content)
		VALUES ($1, $2, $3)
		ON CONFLICT (message_id) DO UPDATE SET
			discord_id = EXCLUDED.discord_id,
			last_content = EXCLUDED.last_content
		`,
		msg.ID,
		discordUser.ID,
		msg.Content, // TODO: Add a method that can fill in mentions and stuff (https://discord.com/developers/docs/reference#message-formatting)
	)

	// Save attachments
	for _, attachment := range msg.Attachments {
		_, err := bot.saveAttachment(ctx, tx, &attachment, discordUser.HMNUserId, msg.ID)
		if err != nil {
			return nil, oops.New(err, "failed to save attachment")
		}
	}

	// Save embeds
	for _, embed := range msg.Embeds {
		_, err := bot.saveEmbed(ctx, tx, &embed, discordUser.HMNUserId, msg.ID)
		if err != nil {
			return nil, oops.New(err, "failed to save embed")
		}
	}

	return newMsg, nil
}

var discordDownloadClient = &http.Client{
	Timeout: 10 * time.Second,
}

type DiscordResourceBadStatusCode error

func downloadDiscordResource(ctx context.Context, url string) ([]byte, string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, "", oops.New(err, "failed to make Discord download request")
	}
	res, err := discordDownloadClient.Do(req)
	if err != nil {
		return nil, "", oops.New(err, "failed to fetch Discord resource data")
	}
	defer res.Body.Close()

	if res.StatusCode < 200 || 299 < res.StatusCode {
		return nil, "", DiscordResourceBadStatusCode(fmt.Errorf("status code %d from Discord resource: %s", res.StatusCode, url))
	}

	content, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	return content, res.Header.Get("Content-Type"), nil
}

func (bot *botInstance) saveAttachment(
	ctx context.Context,
	tx pgx.Tx,
	attachment *Attachment,
	hmnUserID int,
	discordMessageID string,
) (*models.DiscordMessageAttachment, error) {
	// TODO: Return an existing attachment if it exists

	width := 0
	height := 0
	if attachment.Width != nil {
		width = *attachment.Width
	}
	if attachment.Height != nil {
		height = *attachment.Height
	}

	content, _, err := downloadDiscordResource(ctx, attachment.Url)
	if err != nil {
		return nil, oops.New(err, "failed to download Discord attachment")
	}

	contentType := "application/octet-stream"
	if attachment.ContentType != nil {
		contentType = *attachment.ContentType
	}

	asset, err := assets.Create(ctx, tx, assets.CreateInput{
		Content:     content,
		Filename:    attachment.Filename,
		ContentType: contentType,

		UploaderID: &hmnUserID,
		Width:      width,
		Height:     height,
	})
	if err != nil {
		return nil, oops.New(err, "failed to save asset for Discord attachment")
	}

	// TODO(db): RETURNING plz thanks
	_, err = tx.Exec(ctx,
		`
		INSERT INTO handmade_discordmessageattachment (id, asset_id, message_id)
		VALUES ($1, $2, $3)
		`,
		attachment.ID,
		asset.ID,
		discordMessageID,
	)
	if err != nil {
		return nil, oops.New(err, "failed to save Discord attachment data")
	}

	iDiscordAttachment, err := db.QueryOne(ctx, tx, models.DiscordMessageAttachment{},
		`
		SELECT $columns
		FROM handmade_discordmessageattachment
		WHERE id = $1
		`,
		attachment.ID,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch new Discord attachment data")
	}

	return iDiscordAttachment.(*models.DiscordMessageAttachment), nil
}

func (bot *botInstance) saveEmbed(
	ctx context.Context,
	tx pgx.Tx,
	embed *Embed,
	hmnUserID int,
	discordMessageID string,
) (*models.DiscordMessageEmbed, error) {
	// TODO: Does this need to be idempotent

	isOkImageType := func(contentType string) bool {
		return strings.HasPrefix(contentType, "image/")
	}

	isOkVideoType := func(contentType string) bool {
		return strings.HasPrefix(contentType, "video/")
	}

	maybeSaveImageish := func(i EmbedImageish, contentTypeCheck func(string) bool) (*uuid.UUID, error) {
		content, contentType, err := downloadDiscordResource(ctx, *i.Url)
		if err != nil {
			var statusError DiscordResourceBadStatusCode
			if errors.As(err, &statusError) {
				return nil, nil
			} else {
				return nil, oops.New(err, "failed to save Discord embed")
			}
		}
		if contentTypeCheck(contentType) {
			in := assets.CreateInput{
				Content:     content,
				Filename:    "embed",
				ContentType: contentType,
				UploaderID:  &hmnUserID,
			}

			if i.Width != nil {
				in.Width = *i.Width
			}
			if i.Height != nil {
				in.Height = *i.Height
			}

			asset, err := assets.Create(ctx, tx, in)
			if err != nil {
				return nil, oops.New(err, "failed to create asset from embed")
			}
			return &asset.ID, nil
		}

		return nil, nil
	}

	var imageAssetId *uuid.UUID
	var videoAssetId *uuid.UUID
	var err error

	if embed.Video != nil && embed.Video.Url != nil {
		videoAssetId, err = maybeSaveImageish(embed.Video.EmbedImageish, isOkVideoType)
	} else if embed.Image != nil && embed.Image.Url != nil {
		imageAssetId, err = maybeSaveImageish(embed.Image.EmbedImageish, isOkImageType)
	} else if embed.Thumbnail != nil && embed.Thumbnail.Url != nil {
		imageAssetId, err = maybeSaveImageish(embed.Thumbnail.EmbedImageish, isOkImageType)
	}
	if err != nil {
		return nil, err
	}

	// Save the embed into the db
	// TODO(db): Insert, RETURNING
	var savedEmbedId int
	err = tx.QueryRow(ctx,
		`
		INSERT INTO handmade_discordmessageembed (title, description, url, message_id, image_id, video_id)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
		`,
		embed.Title,
		embed.Description,
		embed.Url,
		discordMessageID,
		imageAssetId,
		videoAssetId,
	).Scan(&savedEmbedId)
	if err != nil {
		return nil, oops.New(err, "failed to insert new embed")
	}

	iDiscordEmbed, err := db.QueryOne(ctx, tx, models.DiscordMessageEmbed{},
		`
		SELECT $columns
		FROM handmade_discordmessageembed
		WHERE id = $1
		`,
		savedEmbedId,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch new Discord embed data")
	}

	return iDiscordEmbed.(*models.DiscordMessageEmbed), nil
}

func (bot *botInstance) allowedToCreateMessageSnippet(ctx context.Context, msg *Message) (bool, error) {
	canSave, err := db.QueryBool(ctx, bot.dbConn,
		`
		SELECT u.discord_save_showcase
		FROM
			handmade_discorduser AS duser
			JOIN auth_user AS u ON duser.hmn_user_id = u.id
		WHERE
			duser.userid = $1
		`,
		msg.Author.ID,
	)
	if errors.Is(err, db.ErrNoMatchingRows) {
		return false, nil
	} else if err != nil {
		return false, oops.New(err, "failed to check if we can save Discord message")
	}

	return canSave, nil
}

func (bot *botInstance) createMessageSnippet(ctx context.Context, msg *Message) (*models.Snippet, error) {
	// TODO: Actually do this
	return nil, nil
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

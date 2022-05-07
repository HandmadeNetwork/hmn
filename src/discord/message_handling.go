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
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
	"github.com/google/uuid"
)

func HandleIncomingMessage(ctx context.Context, dbConn db.ConnOrTx, msg *Message, createSnippets bool) error {
	deleted := false
	var err error

	// NOTE(asaf): All functions called here should verify that the message applies to them.

	if !deleted && err == nil {
		deleted, err = CleanUpLibrary(ctx, dbConn, msg)
	}

	if !deleted && err == nil {
		deleted, err = CleanUpShowcase(ctx, dbConn, msg)
	}

	if !deleted && err == nil {
		err = MaybeInternMessage(ctx, dbConn, msg)
	}

	if err == nil {
		err = HandleInternedMessage(ctx, dbConn, msg, deleted, createSnippets)
	}

	return err
}

func CleanUpShowcase(ctx context.Context, dbConn db.ConnOrTx, msg *Message) (bool, error) {
	deleted := false
	if msg.ChannelID == config.Config.Discord.ShowcaseChannelID {
		switch msg.Type {
		case MessageTypeDefault, MessageTypeReply, MessageTypeApplicationCommand:
		default:
			return deleted, nil
		}

		hasGoodContent := true
		if msg.OriginalHasFields("content") && !messageHasLinks(msg.Content) {
			hasGoodContent = false
		}

		hasGoodAttachments := true
		if msg.OriginalHasFields("attachments") && len(msg.Attachments) == 0 {
			hasGoodAttachments = false
		}

		if !hasGoodContent && !hasGoodAttachments {
			err := DeleteMessage(ctx, msg.ChannelID, msg.ID)
			if err != nil {
				return deleted, oops.New(err, "failed to delete message")
			}
			deleted = true

			if !msg.Author.IsBot {
				channel, err := CreateDM(ctx, msg.Author.ID)
				if err != nil {
					return deleted, oops.New(err, "failed to create DM channel")
				}

				err = SendMessages(ctx, dbConn, MessageToSend{
					ChannelID: channel.ID,
					Req: CreateMessageRequest{
						Content: "Posts in #project-showcase are required to have either an image/video or a link. Discuss showcase content in #projects.",
					},
				})
				if err != nil {
					return deleted, oops.New(err, "failed to send showcase warning message")
				}
			}
		}
	}

	return deleted, nil
}

func CleanUpLibrary(ctx context.Context, dbConn db.ConnOrTx, msg *Message) (bool, error) {
	deleted := false
	if msg.ChannelID == config.Config.Discord.LibraryChannelID {
		switch msg.Type {
		case MessageTypeDefault, MessageTypeReply, MessageTypeApplicationCommand:
		default:
			return deleted, nil
		}

		if !msg.OriginalHasFields("content") {
			return deleted, nil
		}

		if !messageHasLinks(msg.Content) {
			err := DeleteMessage(ctx, msg.ChannelID, msg.ID)
			if err != nil {
				return deleted, oops.New(err, "failed to delete message")
			}
			deleted = true

			if !msg.Author.IsBot {
				channel, err := CreateDM(ctx, msg.Author.ID)
				if err != nil {
					return deleted, oops.New(err, "failed to create DM channel")
				}

				err = SendMessages(ctx, dbConn, MessageToSend{
					ChannelID: channel.ID,
					Req: CreateMessageRequest{
						Content: "Posts in #the-library are required to have a link. Discuss library content in other relevant channels.",
					},
				})
				if err != nil {
					return deleted, oops.New(err, "failed to send showcase warning message")
				}
			}
		}
	}

	return deleted, nil
}

func MaybeInternMessage(ctx context.Context, dbConn db.ConnOrTx, msg *Message) error {
	if msg.ChannelID == config.Config.Discord.ShowcaseChannelID {
		err := InternMessage(ctx, dbConn, msg)
		if errors.Is(err, errNotEnoughInfo) {
			logging.ExtractLogger(ctx).Warn().
				Interface("msg", msg).
				Msg("didn't have enough info to intern Discord message")
		} else if err != nil {
			return err
		}
	}
	return nil
}

var errNotEnoughInfo = errors.New("Discord didn't send enough info in this event for us to do this")

/*
Ensures that a Discord message is stored in the database. This function is
idempotent and can be called regardless of whether the item already exists in
the database.

This does not create snippets or save content or do anything besides save the message itself.
*/
func InternMessage(
	ctx context.Context,
	dbConn db.ConnOrTx,
	msg *Message,
) error {
	_, err := db.QueryOne[models.DiscordMessage](ctx, dbConn,
		`
		SELECT $columns
		FROM discord_message
		WHERE id = $1
		`,
		msg.ID,
	)
	if errors.Is(err, db.NotFound) {
		if !msg.OriginalHasFields("author", "timestamp") {
			return errNotEnoughInfo
		}

		guildID := msg.GuildID
		if guildID == nil {
			/*
				This is weird, but it can happen when we fetch messages from
				history instead of receiving it from the gateway. In this case
				we just assume it's from the HMN server.
			*/
			guildID = &config.Config.Discord.GuildID
		}

		_, err = dbConn.Exec(ctx,
			`
			INSERT INTO discord_message (id, channel_id, guild_id, url, user_id, sent_at, snippet_created)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			`,
			msg.ID,
			msg.ChannelID,
			*guildID,
			msg.JumpURL(),
			msg.Author.ID,
			msg.Time(),
			false,
		)
		if err != nil {
			return oops.New(err, "failed to save new discord message")
		}
	} else if err != nil {
		return oops.New(err, "failed to check for existing Discord message")
	}

	return nil
}

type InternedMessage struct {
	Message        models.DiscordMessage         `db:"message"`
	MessageContent *models.DiscordMessageContent `db:"content"`
	HMNUser        *models.User                  `db:"hmnuser"`
	DiscordUser    *models.DiscordUser           `db:"duser"`
}

func FetchInternedMessage(ctx context.Context, dbConn db.ConnOrTx, msgId string) (*InternedMessage, error) {
	interned, err := db.QueryOne[InternedMessage](ctx, dbConn,
		`
		SELECT $columns
		FROM
			discord_message AS message
			LEFT JOIN discord_message_content AS content ON content.message_id = message.id
			LEFT JOIN discord_user AS duser ON duser.userid = message.user_id
			LEFT JOIN hmn_user AS hmnuser ON hmnuser.id = duser.hmn_user_id
			LEFT JOIN asset AS hmnuser_avatar ON hmnuser_avatar.id = hmnuser.avatar_asset_id
		WHERE message.id = $1
		`,
		msgId,
	)
	if err != nil {
		return nil, err
	}
	return interned, nil
}

// Checks if a message is interned and handles it to the extent possible:
// 1. Saves/updates content
// 2. Saves/updates snippet
// 3. Deletes content/snippet
func HandleInternedMessage(ctx context.Context, dbConn db.ConnOrTx, msg *Message, deleted bool, createSnippet bool) error {
	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	interned, err := FetchInternedMessage(ctx, tx, msg.ID)
	if err != nil && !errors.Is(err, db.NotFound) {
		return err
	} else if err == nil {
		if !deleted {
			err = SaveMessageContents(ctx, tx, interned, msg)
			if err != nil {
				return err

			}
			if createSnippet {
				err = HandleSnippetForInternedMessage(ctx, tx, interned, false)
				if err != nil {
					return err
				}
			}
		} else {
			err = DeleteInternedMessage(ctx, tx, interned)
			if err != nil {
				return err
			}
		}
	}
	err = tx.Commit(ctx)
	if err != nil {
		return oops.New(err, "failed to commit Discord message updates")
	}

	return nil
}

func DeleteInternedMessage(ctx context.Context, dbConn db.ConnOrTx, interned *InternedMessage) error {
	snippet, err := db.QueryOne[models.Snippet](ctx, dbConn,
		`
		SELECT $columns
		FROM snippet
		WHERE discord_message_id = $1
		`,
		interned.Message.ID,
	)
	if err != nil && !errors.Is(err, db.NotFound) {
		return oops.New(err, "failed to fetch snippet for discord message")
	}

	// NOTE(asaf): Also deletes the following through a db cascade:
	//			   * discord_message_attachment
	//			   * discord_message_content
	//			   * discord_message_embed
	//             DOES NOT DELETE ASSETS FOR CONTENT/EMBEDS
	_, err = dbConn.Exec(ctx,
		`
		DELETE FROM discord_message
		WHERE id = $1
		`,
		interned.Message.ID,
	)

	if snippet != nil {
		userApprovesDeletion := interned.HMNUser != nil && snippet.OwnerID == interned.HMNUser.ID && interned.HMNUser.DiscordDeleteSnippetOnMessageDelete
		if !snippet.EditedOnWebsite && userApprovesDeletion {
			// NOTE(asaf): Does not delete asset!
			_, err = dbConn.Exec(ctx,
				`
				DELETE FROM snippet
				WHERE id = $1
				`,
				snippet.ID,
			)
			if err != nil {
				return oops.New(err, "failed to delete snippet")
			}
		}
	}

	return nil
}

/*
Processes a single Discord message, saving as much of the message's content
and attachments as allowed by our rules and user settings. Does NOT create
snippets.

Idempotent; can be called any time whether the contents exist or not.

NOTE!!: Replaces interned.MessageContent if it was created or updated!!
*/
func SaveMessageContents(
	ctx context.Context,
	dbConn db.ConnOrTx,
	interned *InternedMessage,
	msg *Message,
) error {
	if interned.DiscordUser != nil {
		// We have a linked Discord account, so save the message contents (regardless of
		// whether we create a snippet or not).
		if msg.OriginalHasFields("content") {
			_, err := dbConn.Exec(ctx,
				`
				INSERT INTO discord_message_content (message_id, discord_id, last_content)
				VALUES ($1, $2, $3)
				ON CONFLICT (message_id) DO UPDATE SET
					discord_id = EXCLUDED.discord_id,
					last_content = EXCLUDED.last_content
				`,
				interned.Message.ID,
				interned.DiscordUser.ID,
				CleanUpMarkdown(ctx, msg.Content),
			)
			if err != nil {
				return oops.New(err, "failed to create or update message contents")
			}

			content, err := db.QueryOne[models.DiscordMessageContent](ctx, dbConn,
				`
				SELECT $columns
				FROM
					discord_message_content
				WHERE
					discord_message_content.message_id = $1
				`,
				interned.Message.ID,
			)
			if err != nil {
				return oops.New(err, "failed to fetch message contents")
			}
			interned.MessageContent = content
		}

		// Save attachments
		if msg.OriginalHasFields("attachments") {
			for _, attachment := range msg.Attachments {
				_, err := saveAttachment(ctx, dbConn, &attachment, interned.DiscordUser.HMNUserId, msg.ID)
				if err != nil {
					return oops.New(err, "failed to save attachment")
				}
			}
		}

		// Save / delete embeds
		if msg.OriginalHasFields("embeds") {
			numSavedEmbeds, err := db.QueryOneScalar[int](ctx, dbConn,
				`
				SELECT COUNT(*)
				FROM discord_message_embed
				WHERE message_id = $1
				`,
				msg.ID,
			)
			if err != nil {
				return oops.New(err, "failed to count existing embeds")
			}
			if numSavedEmbeds == 0 {
				// No embeds yet, so save new ones
				for _, embed := range msg.Embeds {
					_, err := saveEmbed(ctx, dbConn, &embed, interned.DiscordUser.HMNUserId, msg.ID)
					if err != nil {
						return oops.New(err, "failed to save embed")
					}
				}
			} else if len(msg.Embeds) > 0 {
				// Embeds were removed from the message
				_, err := dbConn.Exec(ctx,
					`
				DELETE FROM discord_message_embed
				WHERE message_id = $1
				`,
					msg.ID,
				)
				if err != nil {
					return oops.New(err, "failed to delete embeds")
				}
			}
		}
	}
	return nil
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

/*
Saves a Discord attachment as an HMN asset. Idempotent; will not create an attachment
that already exists
*/
func saveAttachment(
	ctx context.Context,
	tx db.ConnOrTx,
	attachment *Attachment,
	hmnUserID int,
	discordMessageID string,
) (*models.DiscordMessageAttachment, error) {
	existing, err := db.QueryOne[models.DiscordMessageAttachment](ctx, tx,
		`
		SELECT $columns
		FROM discord_message_attachment
		WHERE id = $1
		`,
		attachment.ID,
	)
	if err == nil {
		return existing, nil
	} else if errors.Is(err, db.NotFound) {
		// this is fine, just create it
	} else {
		return nil, oops.New(err, "failed to check for existing attachment")
	}

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
		INSERT INTO discord_message_attachment (id, asset_id, message_id)
		VALUES ($1, $2, $3)
		`,
		attachment.ID,
		asset.ID,
		discordMessageID,
	)
	if err != nil {
		return nil, oops.New(err, "failed to save Discord attachment data")
	}

	discordAttachment, err := db.QueryOne[models.DiscordMessageAttachment](ctx, tx,
		`
		SELECT $columns
		FROM discord_message_attachment
		WHERE id = $1
		`,
		attachment.ID,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch new Discord attachment data")
	}

	return discordAttachment, nil
}

// Saves an embed from Discord. NOTE: This is _not_ idempotent, so only call it
// if you do not have any embeds saved for this message yet.
func saveEmbed(
	ctx context.Context,
	tx db.ConnOrTx,
	embed *Embed,
	hmnUserID int,
	discordMessageID string,
) (*models.DiscordMessageEmbed, error) {
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
		INSERT INTO discord_message_embed (title, description, url, message_id, image_id, video_id)
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

	discordEmbed, err := db.QueryOne[models.DiscordMessageEmbed](ctx, tx,
		`
		SELECT $columns
		FROM discord_message_embed
		WHERE id = $1
		`,
		savedEmbedId,
	)
	if err != nil {
		return nil, oops.New(err, "failed to fetch new Discord embed data")
	}

	return discordEmbed, nil
}

func FetchSnippetForMessage(ctx context.Context, dbConn db.ConnOrTx, msgID string) (*models.Snippet, error) {
	snippet, err := db.QueryOne[models.Snippet](ctx, dbConn,
		`
		SELECT $columns
		FROM snippet
		WHERE discord_message_id = $1
		`,
		msgID,
	)

	if err != nil {
		if errors.Is(err, db.NotFound) {
			return nil, nil
		} else {
			return nil, oops.New(err, "failed to fetch existing snippet for message %s", msgID)
		}
	}

	return snippet, nil
}

/*
Potentially creates or updates a snippet for the given interned message.
It uses the content saved in the database to do this. If we do not have any
content saved, nothing will happen.

If a user does not have their Discord account linked, this function will
naturally do nothing because we have no message content saved.
If forceCreate is true, it does not check any user settings such as automatically creating snippets from
#project-showcase. If we have the content, it will make a snippet for it, no
questions asked. Bear that in mind.
*/
func HandleSnippetForInternedMessage(ctx context.Context, dbConn db.ConnOrTx, interned *InternedMessage, forceCreate bool) error {
	if interned.HMNUser == nil {
		// NOTE(asaf): Can't handle snippets when there's no linked user
		return nil
	}

	if interned.MessageContent == nil {
		// NOTE(asaf): Can't have a snippet without content
		// NOTE(asaf): Messages that only have an attachment also have blank content
		// TODO(asaf): Do we need to delete existing snippets in this case??? Not entirely sure how to trigger this through discord
		return nil
	}

	tx, err := dbConn.Begin(ctx)
	if err != nil {
		oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	existingSnippet, err := FetchSnippetForMessage(ctx, tx, interned.Message.ID)
	if err != nil {
		return oops.New(err, "failed to check for existing snippet for message %s", interned.Message.ID)
	}

	if existingSnippet != nil {
		// TODO(asaf): We're not handling the case where embeds were removed or modified.
		//             Also not handling the case where a message had both an attachment and an embed
		//             and the attachment was removed (leaving only the embed).
		linkedUserIsSnippetOwner := existingSnippet.OwnerID == interned.DiscordUser.HMNUserId
		if linkedUserIsSnippetOwner && !existingSnippet.EditedOnWebsite {
			contentMarkdown := interned.MessageContent.LastContent
			contentHTML := parsing.ParseMarkdown(contentMarkdown, parsing.DiscordMarkdown)

			_, err := tx.Exec(ctx,
				`
				UPDATE snippet
				SET
					description = $1,
					_description_html = $2
				WHERE id = $3
				`,
				contentMarkdown,
				contentHTML,
				existingSnippet.ID,
			)
			if err != nil {
				return oops.New(err, "failed to update content of snippet on message edit")
			}
			existingSnippet.Description = contentMarkdown
			existingSnippet.DescriptionHtml = contentHTML
		}
	} else {
		userAllowsSnippet := interned.HMNUser.DiscordSaveShowcase || forceCreate
		shouldCreate := !interned.Message.SnippetCreated && userAllowsSnippet

		if shouldCreate {
			// Get an asset ID or URL to make a snippet from
			assetId, url, err := getSnippetAssetOrUrl(ctx, tx, &interned.Message)
			if assetId != nil || url != nil {
				contentMarkdown := interned.MessageContent.LastContent
				contentHTML := parsing.ParseMarkdown(contentMarkdown, parsing.DiscordMarkdown)

				_, err = tx.Exec(ctx,
					`
					INSERT INTO snippet (url, "when", description, _description_html, asset_id, discord_message_id, owner_id)
					VALUES ($1, $2, $3, $4, $5, $6, $7)
					`,
					url,
					interned.Message.SentAt,
					contentMarkdown,
					contentHTML,
					assetId,
					interned.Message.ID,
					interned.HMNUser.ID,
				)
				if err != nil {
					return oops.New(err, "failed to create snippet from attachment")
				}

				existingSnippet, err = FetchSnippetForMessage(ctx, tx, interned.Message.ID)
				if err != nil {
					return oops.New(err, "failed to fetch newly-created snippet")
				}

				_, err = tx.Exec(ctx,
					`
					UPDATE discord_message
					SET snippet_created = TRUE
					WHERE id = $1
					`,
					interned.Message.ID,
				)
				if err != nil {
					return oops.New(err, "failed to mark message as having snippet")
				}
			}
		}
	}

	if existingSnippet != nil {
		// Update tags

		// Try to associate tags in the message with project tags in HMN.
		// Match only tags for projects in which the current user is a collaborator.
		messageTags := getDiscordTags(existingSnippet.Description)

		var desiredTags []int
		var allTags []int

		// Fetch projects so we know what tags the user can apply to their snippet.
		projects, err := hmndata.FetchProjects(ctx, tx, interned.HMNUser, hmndata.ProjectsQuery{
			OwnerIDs: []int{interned.HMNUser.ID},
		})
		if err != nil {
			return oops.New(err, "failed to look up user projects")
		}

		projectIDs := make([]int, len(projects))
		for i, p := range projects {
			projectIDs[i] = p.Project.ID
		}

		userTags, err := db.Query[models.Tag](ctx, tx,
			`
			SELECT $columns{tag}
			FROM
				tags
				JOIN project ON project.tag = tag.id
			WHERE
				project.id = ANY ($1)
			`,
			projectIDs,
		)
		if err != nil {
			return oops.New(err, "failed to fetch tags for user projects")
		}

		for _, tag := range userTags {
			allTags = append(allTags, tag.ID)
			for _, messageTag := range messageTags {
				if strings.EqualFold(tag.Text, messageTag) {
					desiredTags = append(desiredTags, tag.ID)
				}
			}
		}

		_, err = tx.Exec(ctx,
			`
			DELETE FROM snippet_tag
			WHERE
				snippet_id = $1
				AND tag_id = ANY ($2)
			`,
			existingSnippet.ID,
			allTags,
		)
		if err != nil {
			return oops.New(err, "failed to clear tags from snippet")
		}

		for _, tagID := range desiredTags {
			_, err = tx.Exec(ctx,
				`
				INSERT INTO snippet_tag (snippet_id, tag_id)
				VALUES ($1, $2)
				ON CONFLICT DO NOTHING
				`,
				existingSnippet.ID,
				tagID,
			)
			if err != nil {
				return oops.New(err, "failed to associate snippet with tag")
			}
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return oops.New(err, "failed to commit transaction")
	}

	return nil
}

// TODO(asaf): I believe this will also match https://example.com?hello=1&whatever=5
//             Probably need to add word boundaries.
var REDiscordTag = regexp.MustCompile(`&([a-zA-Z0-9]+(-[a-zA-Z0-9]+)*)`)

func getDiscordTags(content string) []string {
	matches := REDiscordTag.FindAllStringSubmatch(content, -1)
	result := make([]string, len(matches))
	for i, m := range matches {
		result[i] = m[1]
	}
	return result
}

// NOTE(ben): This is maybe redundant with the regexes we use for markdown. But
// do we actually want to reuse those, or should we keep them separate?
var RESnippetableUrl = regexp.MustCompile(`^https?://(youtu\.be|(www\.)?youtube\.com/watch)`)

func getSnippetAssetOrUrl(ctx context.Context, tx db.ConnOrTx, msg *models.DiscordMessage) (*uuid.UUID, *string, error) {
	// Check attachments
	attachments, err := db.Query[models.DiscordMessageAttachment](ctx, tx,
		`
		SELECT $columns
		FROM discord_message_attachment
		WHERE message_id = $1
		`,
		msg.ID,
	)
	if err != nil {
		return nil, nil, oops.New(err, "failed to fetch message attachments")
	}
	for _, attachment := range attachments {
		return &attachment.AssetID, nil, nil
	}

	// Check embeds
	embeds, err := db.Query[models.DiscordMessageEmbed](ctx, tx,
		`
		SELECT $columns
		FROM discord_message_embed
		WHERE message_id = $1
		`,
		msg.ID,
	)
	if err != nil {
		return nil, nil, oops.New(err, "failed to fetch discord embeds")
	}
	for _, embed := range embeds {
		if embed.VideoID != nil {
			return embed.VideoID, nil, nil
		} else if embed.ImageID != nil {
			return embed.ImageID, nil, nil
		} else if embed.URL != nil {
			if RESnippetableUrl.MatchString(*embed.URL) {
				return nil, embed.URL, nil
			}
		}
	}

	return nil, nil, nil
}

var reDiscordMessageLink = regexp.MustCompile(`https?://.+?(\s|$)`)

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

package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/parsing"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/google/uuid"
)

var autostoreChannels = []string{
	config.Config.Discord.ShowcaseChannelID,
	// TODO(asaf): Add jam channel
}

func HandleIncomingMessage(ctx context.Context, dbConn db.ConnOrTx, msg *Message, notifyUser bool) error {
	deleted := false
	interned := false
	var err error

	// NOTE(asaf): All functions called here should verify that the message applies to them.

	if !deleted && err == nil {
		deleted, err = CleanUpLibrary(ctx, dbConn, msg)
	}

	// if !deleted && err == nil {
	// 	err = ShareToMatrix(ctx, msg)
	// }

	tags := parseTags(msg.Content)

	if !deleted && err == nil {
		interned, err = MaybeInternMessage(ctx, dbConn, msg, tags)
	}

	if !deleted && err == nil {
		deleted, err = CleanUpShowcase(ctx, dbConn, msg, interned)
	}

	if err == nil {
		err = HandleInternedMessage(ctx, dbConn, msg, deleted, true, notifyUser)
	}

	// when we needed her most...she vanished
	// if !deleted && err == nil {
	// 	err = FreyaMode(ctx, dbConn, msg)
	// }

	return err
}

type TagProject struct {
	Tag       string `db:"tag.text"`
	ProjectID int    `db:"project.id"`
}

func HandleIncomingMessageTags(ctx context.Context, dbConn db.ConnOrTx, hmnUser *models.User, authorID string, tags []string, notifyUser bool) ([]*TagProject, error) {
	if len(tags) > 0 && hmnUser != nil {
		existingUserTags, err := db.Query[TagProject](ctx, dbConn,
			`
			SELECT $columns
			FROM
				tag
				JOIN project ON project.tag = tag.id
				JOIN user_project ON user_project.project_id = project.id
			WHERE user_project.user_id = $1 AND tag.text = ANY ($2)
			`,
			hmnUser.ID,
			tags,
		)
		if err != nil {
			return nil, err
		}

		missingTags := []string{}
		for _, t := range tags {
			found := false
			for _, et := range existingUserTags {
				if et.Tag == t {
					found = true
					break
				}
			}
			if !found {
				missingTags = append(missingTags, t)
			}
		}

		if len(missingTags) > 0 && notifyUser {
			err = SendDM(ctx, dbConn,
				authorID,
				fmt.Sprintf(
					"It looks like you tried linking a message to a Handmade Network project, but we couldn't find a project matching the tags you provided (%s).\nGo to your project's settings on the Handmade Network website, set a Discord tag, then edit or repost your message on Discord.\nIf this is a false positive, please let us know in #network-meta.",
					strings.Join(missingTags, ", "),
				),
			)
			if err != nil {
				return existingUserTags, oops.New(err, "failed to send unidentified tags warning message")
			}
		}
		return existingUserTags, nil
	}

	return nil, nil
}

func CleanUpShowcase(ctx context.Context, dbConn db.ConnOrTx, msg *Message, interned bool) (bool, error) {
	deleted := false
	if msg.ChannelID == config.Config.Discord.ShowcaseChannelID {
		switch msg.Type {
		case MessageTypeDefault, MessageTypeReply, MessageTypeApplicationCommand:
		default:
			return deleted, nil
		}

		if !interned {
			err := DeleteMessage(ctx, msg.ChannelID, msg.ID)
			if err != nil {
				return deleted, oops.New(err, "failed to delete message")
			}
			deleted = true

			if !msg.Author.IsBot {
				err = SendDM(ctx, dbConn, msg.Author.ID,
					"Posts in #project-showcase are required to have either an image/video or a link, or start with `!til`, or be tagged to an hmn project (like `&myproject`). Discuss showcase content in #project-discussion.",
				)

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
				err = SendDM(ctx, dbConn, msg.Author.ID,
					"Posts in #the-library are required to have a link. Discuss library content in other relevant channels.",
				)
				if err != nil {
					return deleted, oops.New(err, "failed to send showcase warning message")
				}
			}
		}
	}

	return deleted, nil
}

func FreyaMode(ctx context.Context, dbConn db.ConnOrTx, msg *Message) error {
	if msg.Author.IsBot {
		return nil
	}
	if msg.ChannelID == config.Config.Discord.ShowcaseChannelID || msg.ChannelID == config.Config.Discord.LibraryChannelID {
		return nil
	}

	twitteryUrls := []string{
		"https://twitter.com",
		"https://x.com",
		"https://vxtwitter.com",
		"https://fxtwitter.com",
	}
	isTwittery := false
	for _, url := range twitteryUrls {
		if strings.Contains(msg.Content, url) {
			isTwittery = true
		}
	}
	if !isTwittery {
		return nil
	}

	// FREYA MODE ENGAGED
	approvedTweets := []string{
		"https://vxtwitter.com/FreyaHolmer/status/1757836988495847568",
		"https://vxtwitter.com/FreyaHolmer/status/1752441092501361103",
		"https://vxtwitter.com/FreyaHolmer/status/1753813557966217268",
		"https://vxtwitter.com/FreyaHolmer/status/1746228932188295579",
		"https://vxtwitter.com/FreyaHolmer/status/1732687685850894799",
		"https://vxtwitter.com/FreyaHolmer/status/1761487879178736048",
		"https://vxtwitter.com/FreyaHolmer/status/1733820461492863442",
		"https://vxtwitter.com/FreyaHolmer/status/1732845451701871101",
		"https://vxtwitter.com/FreyaHolmer/status/1765680355657359585",
		"https://vxtwitter.com/FreyaHolmer/status/1784678195997852129",
		"https://vxtwitter.com/FreyaHolmer/status/1741468609044508831",
		"https://vxtwitter.com/FreyaHolmer/status/1759306434053870012",
		"https://vxtwitter.com/FreyaHolmer/status/1754929898492162178",
		"https://vxtwitter.com/FreyaHolmer/status/1782498313511534822",
		"https://vxtwitter.com/FreyaHolmer/status/1623737764041695232",
		"https://vxtwitter.com/FreyaHolmer/status/1718979996125925494",
		"https://vxtwitter.com/FreyaHolmer/status/1675945798448607248",
		"https://vxtwitter.com/FreyaHolmer/status/1662229911375953922",
		"https://vxtwitter.com/FreyaHolmer/status/1652235944752185345",
		"https://vxtwitter.com/FreyaHolmer/status/1386408507218427905",
		"https://vxtwitter.com/FreyaHolmer/status/1436696408506212353",
		"https://vxtwitter.com/FreyaHolmer/status/1444755552777670657",
		"https://vxtwitter.com/FreyaHolmer/status/1232826293902888960",
	}
	tweet := approvedTweets[rand.Intn(len(approvedTweets))]
	err := SendMessages(ctx, dbConn, MessageToSend{
		ChannelID: msg.ChannelID,
		Req: CreateMessageRequest{
			Content: fmt.Sprintf("No. Only Freya is allowed to tweet. %s", tweet),
		},
	})
	if err != nil {
		return oops.New(err, "failed to send Freya tweet")
	}

	return nil
}

func ShareToMatrix(ctx context.Context, msg *Message) error {
	if msg.Flags&MessageFlagCrossposted == 0 {
		return nil
	}
	if config.Config.Matrix.Username == "" {
		logging.ExtractLogger(ctx).Warn().Msg("No Matrix user provided; Discord announcement will not be shared")
	}

	fullMsg, err := GetChannelMessage(ctx, msg.ChannelID, msg.ID)
	if err != nil {
		return oops.New(err, "failed to get published message contents")
	}

	bodyMarkdown := CleanUpMarkdown(ctx, fullMsg.Content)
	bodyHTML := parsing.ParseMarkdown(bodyMarkdown, parsing.DiscordMarkdown)

	// Log in to Matrix (we don't bother to keep access tokens around)
	var accessToken string
	{
		type MatrixLogin struct {
			Type     string `json:"type"`
			User     string `json:"user"`
			Password string `json:"password"`
		}
		type MatrixLoginResponse struct {
			AccessToken string `json:"access_token"`
		}
		body := MatrixLogin{
			Type:     "m.login.password",
			User:     config.Config.Matrix.Username,
			Password: config.Config.Matrix.Password,
		}
		bodyBytes := utils.Must1(json.Marshal(body))
		res, err := http.Post(
			"https://matrix.handmadecities.com/_matrix/client/r0/login",
			"application/json",
			bytes.NewReader(bodyBytes),
		)
		if err != nil || res.StatusCode >= 300 {
			return oops.New(err, "failed to log into Matrix")
		}
		defer res.Body.Close()
		resBodyBytes := utils.Must1(io.ReadAll(res.Body))
		var resBody MatrixLoginResponse
		utils.Must(json.Unmarshal(resBodyBytes, &resBody))

		accessToken = resBody.AccessToken
	}

	// Create message
	{
		type MessageEvent struct {
			MsgType       string `json:"msgtype"`
			Body          string `json:"body"`
			Format        string `json:"format,omitempty"`
			FormattedBody string `json:"formatted_body,omitempty"`
		}
		tid := "hmn" + strconv.Itoa(rand.Int())
		body := MessageEvent{
			MsgType:       "m.text",
			Body:          bodyMarkdown,
			Format:        "org.matrix.custom.html",
			FormattedBody: bodyHTML,
		}
		bodyBytes := utils.Must1(json.Marshal(body))
		req := utils.Must1(http.NewRequestWithContext(
			ctx,
			http.MethodPut,
			fmt.Sprintf(
				"%s/_matrix/client/v3/rooms/%s/send/m.room.message/%s",
				config.Config.Matrix.BaseUrl,
				config.Config.Matrix.AnnouncementsRoomID,
				tid,
			),
			bytes.NewReader(bodyBytes),
		))
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		req.Header.Add("Content-Type", "application/json")
		res, err := http.DefaultClient.Do(req)
		if err != nil || res.StatusCode >= 300 {
			return oops.New(err, "failed to send Matrix message")
		}
	}

	logging.ExtractLogger(ctx).Info().
		Str("contents", bodyMarkdown).
		Msg("Published Discord announcement to Matrix")

	return nil
}

func MaybeInternMessage(ctx context.Context, dbConn db.ConnOrTx, msg *Message, tags []string) (bool, error) {
	interned := false
	if messageShouldBeStored(msg, tags) {
		interned = true
		err := InternMessage(ctx, dbConn, msg)
		if errors.Is(err, errNotEnoughInfo) {
			logging.ExtractLogger(ctx).Warn().
				Interface("msg", msg).
				Msg("didn't have enough info to intern Discord message")
		} else if err != nil {
			return interned, err
		}
	}
	return interned, nil
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
			INSERT INTO discord_message (id, channel_id, guild_id, url, user_id, sent_at, snippet_created, backfilled)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			`,
			msg.ID,
			msg.ChannelID,
			*guildID,
			msg.JumpURL(),
			msg.Author.ID,
			msg.Time(),
			false,
			msg.Backfilled,
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
		WHERE message.id = $1
		`,
		msgId,
	)
	if err != nil {
		return nil, err
	}
	if interned.MessageContent != nil {
		interned.MessageContent.ParsedTags = parseTags(interned.MessageContent.LastContent)
	}
	return interned, nil
}

// Checks if a message is interned and handles it to the extent possible:
// 1. Saves/updates content
// 2. Saves/updates snippet
// 3. Deletes content/snippet
func HandleInternedMessage(ctx context.Context, dbConn db.ConnOrTx, msg *Message, messageDeleted bool, createSnippet bool, notifyUser bool) error {
	tx, err := dbConn.Begin(ctx)
	if err != nil {
		return oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	interned, err := FetchInternedMessage(ctx, tx, msg.ID)
	if err != nil && !errors.Is(err, db.NotFound) {
		return err
	} else if err == nil {
		if !messageDeleted {
			if msg.InternType == InternTypeExplicit && interned.DiscordUser == nil {
				if notifyUser {
					err := SendDM(ctx, dbConn,
						msg.Author.ID,
						fmt.Sprintf("It looks like you tried linking a message to a Handmade Network project, but your Discord account is not linked to your Handmade Network account.\nYou can link your account at <%s>, then edit or repost your message on Discord.\nIf this is a false positive, please let us know in #network-meta.", hmnurl.BuildUserSettings("discord")),
					)
					if err != nil {
						return oops.New(err, "failed to send unlinked account warning message")
					}
				}
			}
			err = SaveMessageContents(ctx, tx, interned, msg)
			if err != nil {
				return err

			}
			if createSnippet {
				err = HandleSnippetForInternedMessage(ctx, tx, interned, false, notifyUser)
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
			interned.MessageContent.ParsedTags = parseTags(interned.MessageContent.LastContent)
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

func DownloadDiscordResource(ctx context.Context, url string) ([]byte, string, error) {
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
		logging.ExtractLogger(ctx).Error().Str("Url", url).Msg("failed to download resource")
		return nil, "", err
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

	content, _, err := DownloadDiscordResource(ctx, attachment.Url)
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
		content, contentType, err := DownloadDiscordResource(ctx, *i.Url)
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
func HandleSnippetForInternedMessage(ctx context.Context, dbConn db.ConnOrTx, interned *InternedMessage, forceCreate bool, notifyUser bool) error {
	if interned.HMNUser == nil {
		// NOTE(asaf): Can't handle snippets when there's no linked user
		return nil
	}

	if interned.MessageContent == nil {
		// NOTE(asaf): Can't have a snippet without content
		// NOTE(asaf): Messages that only have an attachment also have a content struct with an empty content string
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

	if existingSnippet != nil {
		// Update tags

		// Try to associate tags in the message with project tags in HMN.
		// Match only tags for projects in which the current user is a collaborator.
		tags, err := HandleIncomingMessageTags(ctx, tx, interned.HMNUser, interned.Message.UserID, interned.MessageContent.ParsedTags, notifyUser)
		if err != nil {
			return err
		}

		_, err = tx.Exec(ctx,
			`
			DELETE FROM snippet_project
			WHERE
				snippet_id = $1
				AND kind = $2
			`,
			existingSnippet.ID,
			models.SnippetProjectKindDiscord,
		)
		if err != nil {
			return oops.New(err, "failed to clear project association for snippet")
		}

		for _, t := range tags {
			_, err = tx.Exec(ctx,
				`
				INSERT INTO snippet_project (project_id, snippet_id, kind)
				VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING
				`,
				t.ProjectID,
				existingSnippet.ID,
				models.SnippetProjectKindDiscord,
			)
			if err != nil {
				return oops.New(err, "failed to associate snippet with project")
			}

		}

		hmndata.UpdateSnippetLastPostedForAllProjects(ctx, tx)
	}

	err = tx.Commit(ctx)
	if err != nil {
		return oops.New(err, "failed to commit transaction")
	}

	return nil
}

// NOTE(ben): This is maybe redundant with the regexes we use for markdown. But
// do we actually want to reuse those, or should we keep them separate?
// TODO(asaf): Centralize this
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

func messageShouldBeStored(msg *Message, tags []string) bool {
	if msg == nil {
		return false
	}

	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(msg.Content)), "!til") {
		msg.InternType = InternTypeExplicit
		return true
	}

	if len(tags) > 0 {
		msg.InternType = InternTypeExplicit
		return true
	}

	autostore := false
	for _, cid := range autostoreChannels {
		if msg.ChannelID == cid {
			autostore = true
			break
		}
	}
	if autostore {
		hasGoodContent := true
		if msg.OriginalHasFields("content") && !messageHasLinks(msg.Content) {
			hasGoodContent = false
		}

		hasGoodAttachments := true
		if msg.OriginalHasFields("attachments") && len(msg.Attachments) == 0 {
			hasGoodAttachments = false
		}

		intern := hasGoodContent || hasGoodAttachments
		if intern {
			msg.InternType = InternTypeImplicit
		}

		return intern
	}

	return false
}

func parseTags(content string) []string {
	var tags []string

	tagStr := parsing.ParseMarkdown(content, parsing.DiscordTagMarkdown)
	tagStr = strings.Trim(tagStr, "\n")
	if len(tagStr) > 0 {
		tags = strings.Split(tagStr, "\n")
	}

	for idx, tag := range tags {
		tags[idx] = strings.ToLower(tag)
	}

	logging.Debug().Strs("tags", tags).Msg("tags")

	return tags
}

func SendDM(ctx context.Context, dbConn db.ConnOrTx, authorID string, text string) error {
	channel, err := CreateDM(ctx, authorID)
	if err != nil {
		return oops.New(err, "failed to create DM channel")
	}

	err = SendMessages(ctx, dbConn, MessageToSend{
		ChannelID: channel.ID,
		Req: CreateMessageRequest{
			Content: text,
		},
	})
	if err != nil {
		return oops.New(err, "failed to send DM to message owner")
	}

	return nil
}

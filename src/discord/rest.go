package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"os"
	"strconv"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
)

const (
	BotName = "HandmadeNetwork"
	BaseUrl = "https://discord.com/api/v9"

	UserAgentURL     = "https://handmade.network/"
	UserAgentVersion = "1.0"
)

var UserAgent = fmt.Sprintf("%s (%s, %s)", BotName, UserAgentURL, UserAgentVersion)

var NotFound = errors.New("not found")

var httpClient = &http.Client{}

func buildUrl(path string) string {
	return fmt.Sprintf("%s%s", BaseUrl, path)
}

// NOTE(ben): Creates an [http.Request] to send to Discord, with the correct
// auth and user-agent headers. Also attaches the given body to the request. In
// many cases the request will need to be modified further, e.g. setting
// Content-Type or query params.
func createTypicalRequest(ctx context.Context, method string, path string, body []byte) *http.Request {
	var bodyReader io.Reader
	if body != nil {
		bodyReader = bytes.NewBuffer(body)
	}

	req, err := http.NewRequestWithContext(ctx, method, buildUrl(path), bodyReader)
	if err != nil {
		panic(err)
	}
	req.Header.Add("Authorization", fmt.Sprintf("Bot %s", config.Config.Discord.BotToken))
	req.Header.Add("User-Agent", UserAgent)

	return req
}

// NOTE(ben): Performs the given request with rate limiting and returns the
// response + the full response body. Returns an error if the status code is
// not in the OK range. 404s are always turned into [NotFound] errors.
//
// If you want to read the response as JSON, use [unmarshal] with the results.
//
// In most cases you can use [doSimpleGetRequest] for GET,
// [doSimplePostishRequest] for POST or PATCH etc., and [doSimpleDeleteRequest]
// for DELETE.
func doRequest(ctx context.Context, name string, getReq func(context.Context) *http.Request) (context.Context, *http.Response, []byte, error) {
	l := logging.ExtractLogger(ctx).With().Str("name", name).Logger()
	ctx = logging.AttachLoggerToContext(&l, ctx)

	res, err := doWithRateLimiting(ctx, name, getReq)
	if err != nil {
		return ctx, nil, nil, err
	}
	defer res.Body.Close()

	bodyBytes := utils.Must1(io.ReadAll(res.Body))
	if res.StatusCode == http.StatusNotFound {
		return ctx, nil, nil, NotFound
	} else if res.StatusCode >= 400 {
		logErrorResponse(ctx, res, bodyBytes, "")
		return ctx, nil, nil, oops.New(nil, "received error from Discord")
	}

	return ctx, res, bodyBytes, nil
}

// NOTE(ben): Takes the result of [doRequest] and unmarshals the result
// as JSON.
func unmarshal[T any](_ context.Context, _ *http.Response, body []byte, err error) (T, error) {
	var zero T
	if err != nil {
		return zero, err
	}

	var payload T
	err = json.Unmarshal(body, &payload)
	if err != nil {
		return zero, oops.New(err, "failed to unmarshal Discord message")
	}
	return payload, nil
}

// NOTE(ben): Takes the result of [doRequest] and discards everything but the
// error.
func discard(_ context.Context, _ *http.Response, _ []byte, err error) error {
	return err
}

// NOTE(ben): Takes the result of [doRequest] and asserts that the result
// was a 204 No Content.
func expectNoContent(ctx context.Context, res *http.Response, body []byte, err error) error {
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusNoContent {
		logErrorResponse(ctx, res, body, "")
		return oops.New(nil, "expected 204 but got %d", res.StatusCode)
	}
	return nil
}

// NOTE(ben): Does a GET request to the given path and unmarshals the result.
func doSimpleGetRequest[T any](ctx context.Context, name, path string) (T, error) {
	return unmarshal[T](doRequest(ctx, name, func(ctx context.Context) *http.Request {
		return createTypicalRequest(ctx, http.MethodGet, path, nil)
	}))
}

// NOTE(ben): Does a request with a method, content-type, and body of your
// choice. Designed to work for POST, PATCH, and similar where you don't need
// to provide other headers. Does not do anything with the result, so consider
// using [unmarshal], [expectNoContent], etc.
func doSimplePostishRequest(ctx context.Context, name, method, path, contentType string, body []byte) (context.Context, *http.Response, []byte, error) {
	return doRequest(ctx, name, func(ctx context.Context) *http.Request {
		req := createTypicalRequest(ctx, method, path, body)
		req.Header.Add("Content-Type", contentType)
		return req
	})
}

// NOTE(ben): Does a DELETE request to the given path and expects a 204.
func doSimpleDeleteRequest(ctx context.Context, name, path string) error {
	return expectNoContent(doRequest(ctx, name, func(ctx context.Context) *http.Request {
		return createTypicalRequest(ctx, http.MethodDelete, path, nil)
	}))
}

type GetGatewayBotResponse struct {
	URL string `json:"url"`
	// We don't care about shards or session limit stuff; we will never hit those limits
}

// https://docs.discord.com/developers/events/gateway#get-gateway-bot
func GetGatewayBot(ctx context.Context) (GetGatewayBotResponse, error) {
	return doSimpleGetRequest[GetGatewayBotResponse](ctx, "Get Gateway Bot", "/gateway/bot")
}

// https://docs.discord.com/developers/resources/guild#get-guild-roles
func GetGuildRoles(ctx context.Context, guildID string) ([]Role, error) {
	path := fmt.Sprintf("/guilds/%s/roles", guildID)
	return doSimpleGetRequest[[]Role](ctx, "Get Guild Roles", path)
}

// https://docs.discord.com/developers/resources/guild#get-guild-channels
func GetGuildChannels(ctx context.Context, guildID string) ([]Channel, error) {
	path := fmt.Sprintf("/guilds/%s/channels", guildID)
	return doSimpleGetRequest[[]Channel](ctx, "Get Guild Channels", path)
}

// https://docs.discord.com/developers/resources/guild#get-guild-member
func GetGuildMember(ctx context.Context, guildID, userID string) (GuildMember, error) {
	path := fmt.Sprintf("/guilds/%s/members/%s", guildID, userID)
	return doSimpleGetRequest[GuildMember](ctx, "Get Guild Member", path)
}

// WARNING: This function is very expensive, as it must make several paginated requests. It will
// block while doing so. It will also allocate memory for all of the guild members too. Please
// consider whether you actually need to do this, and take appropriate precautions if you must.
func ListGuildMembers(ctx context.Context, guildID string) ([]GuildMember, error) {
	path := fmt.Sprintf("/guilds/%s/members", guildID)
	const limit = 1000

	var allMembers []GuildMember
	var lastID string
	for {
		msg, err := unmarshal[[]GuildMember](doRequest(ctx, "List Guild Members", func(ctx context.Context) *http.Request {
			req := createTypicalRequest(ctx, http.MethodGet, path, nil)
			q := req.URL.Query()
			q.Add("limit", strconv.Itoa(limit))
			if lastID != "" {
				q.Add("after", lastID)
			}
			req.URL.RawQuery = q.Encode()

			return req
		}))
		if err != nil {
			return nil, err
		}

		if len(msg) > 0 {
			lastMember := msg[len(msg)-1]
			utils.Assert(lastMember.User, "all guild members from this endpoint should have users")
			lastID = lastMember.User.ID
		}
		allMembers = append(allMembers, msg...)

		if len(msg) < limit {
			break
		}
	}

	return allMembers, nil
}

type MentionType string

const (
	MentionTypeUsers    MentionType = "users"
	MentionTypeRoles    MentionType = "roles"
	MentionTypeEveryone MentionType = "everyone"
)

type MessageAllowedMentions struct {
	Parse []MentionType `json:"parse"`
}

const (
	FlagSuppressEmbeds int = 1 << 2
)

type CreateMessageRequest struct {
	Content         string                  `json:"content"`
	Flags           int                     `json:"flags,omitempty"`
	AllowedMentions *MessageAllowedMentions `json:"allowed_mentions,omitempty"`
}

// https://docs.discord.com/developers/resources/message#create-message
func CreateMessage(ctx context.Context, channelID string, payloadJSON string, files ...FileUpload) (Message, error) {
	path := fmt.Sprintf("/channels/%s/messages", channelID)
	contentType, body := makeNewMessageBody(payloadJSON, files)
	return unmarshal[Message](doSimplePostishRequest(ctx, "Create Message", http.MethodPost, path, contentType, body))

	// Maybe in the future we could more nicely handle errors like "bad channel",
	// but honestly what are the odds that we mess that up...
}

// https://docs.discord.com/developers/resources/message#edit-message
func EditMessage(ctx context.Context, channelID string, messageID string, payloadJSON string, files ...FileUpload) (Message, error) {
	path := fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID)
	contentType, body := makeNewMessageBody(payloadJSON, files)
	return unmarshal[Message](doSimplePostishRequest(ctx, "Edit Message", http.MethodPatch, path, contentType, body))
}

// https://docs.discord.com/developers/resources/message#delete-message
func DeleteMessage(ctx context.Context, channelID string, messageID string) error {
	path := fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID)
	return doSimpleDeleteRequest(ctx, "Delete Message", path)
}

// https://docs.discord.com/developers/resources/user#create-dm
func CreateDM(ctx context.Context, recipientID string) (Channel, error) {
	path := "/users/@me/channels"
	body := fmt.Appendf(nil, `{"recipient_id":"%s"}`, recipientID)
	return unmarshal[Channel](doSimplePostishRequest(ctx, "Create DM", http.MethodPost, path, "application/json", body))
}

type OAuthCodeExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func ExchangeOAuthCode(ctx context.Context, code, redirectURI string) (OAuthCodeExchangeResponse, error) {
	body := make(url.Values)
	body.Set("client_id", config.Config.Discord.OAuthClientID)
	body.Set("client_secret", config.Config.Discord.OAuthClientSecret)
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("redirect_uri", redirectURI)
	bodyStr := body.Encode()

	return unmarshal[OAuthCodeExchangeResponse](doSimplePostishRequest(ctx,
		"OAuth Code Exchange",
		http.MethodPost, "/oauth2/token",
		"application/x-www-form-urlencoded", []byte(bodyStr),
	))
}

// https://docs.discord.com/developers/resources/user#get-current-user
func GetCurrentUserAsOAuth(ctx context.Context, accessToken string) (User, error) {
	return unmarshal[User](doRequest(ctx, "Get Current User", func(ctx context.Context) *http.Request {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildUrl("/users/@me"), nil)
		if err != nil {
			panic(err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		req.Header.Add("User-Agent", UserAgent)

		return req
	}))
}

// https://docs.discord.com/developers/resources/guild#add-guild-member-role
func AddGuildMemberRole(ctx context.Context, userID, roleID, reason string) error {
	path := fmt.Sprintf("/guilds/%s/members/%s/roles/%s", config.Config.Discord.GuildID, userID, roleID)
	return expectNoContent(doRequest(ctx, "Add Guild Member Role", func(ctx context.Context) *http.Request {
		req := createTypicalRequest(ctx, http.MethodPut, path, nil)
		req.Header.Add("X-Audit-Log-Reason", reason)
		return req
	}))
}

// https://docs.discord.com/developers/resources/guild#remove-guild-member-role
func RemoveGuildMemberRole(ctx context.Context, userID, roleID, reason string) error {
	path := fmt.Sprintf("/guilds/%s/members/%s/roles/%s", config.Config.Discord.GuildID, userID, roleID)
	return expectNoContent(doRequest(ctx, "Remove Guild Member Role", func(ctx context.Context) *http.Request {
		req := createTypicalRequest(ctx, http.MethodDelete, path, nil)
		req.Header.Add("X-Audit-Log-Reason", reason)
		return req
	}))
}

// https://docs.discord.com/developers/resources/message#get-channel-message
func GetChannelMessage(ctx context.Context, channelID, messageID string) (Message, error) {
	path := fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID)
	return doSimpleGetRequest[Message](ctx, "Get Channel Message", path)
}

type GetChannelMessagesInput struct {
	Around string
	Before string
	After  string
	Limit  int
}

// https://docs.discord.com/developers/resources/message#get-channel-messages
func GetChannelMessages(ctx context.Context, channelID string, in GetChannelMessagesInput) ([]Message, error) {
	path := fmt.Sprintf("/channels/%s/messages", channelID)
	return unmarshal[[]Message](doRequest(ctx, "Get Channel Messages", func(ctx context.Context) *http.Request {
		req := createTypicalRequest(ctx, http.MethodGet, path, nil)
		q := req.URL.Query()
		if in.Around != "" {
			q.Add("around", in.Around)
		}
		if in.Before != "" {
			q.Add("before", in.Before)
		}
		if in.After != "" {
			q.Add("after", in.After)
		}
		if in.Limit != 0 {
			q.Add("limit", strconv.Itoa(in.Limit))
		}
		req.URL.RawQuery = q.Encode()

		return req
	}))
}

// See https://discord.com/developers/docs/interactions/application-commands#create-guild-application-command-json-params
type CreateGuildApplicationCommandRequest struct {
	Name              string                     `json:"name"`               // 1-32 character name
	Description       string                     `json:"description"`        // 1-100 character description
	Options           []ApplicationCommandOption `json:"options"`            // the parameters for the command
	DefaultPermission *bool                      `json:"default_permission"` // whether the command is enabled by default when the app is added to a guild
	Type              ApplicationCommandType     `json:"type"`               // the type of command, defaults 1 if not set
	DMPermission      *bool                      `json:"dm_permission"`      // Technically deprecated, but needs to be set to false in order to get member info in the interaction object
}

// See https://discord.com/developers/docs/interactions/application-commands#create-guild-application-command
func CreateGuildApplicationCommand(ctx context.Context, in CreateGuildApplicationCommandRequest) error {
	if in.Type == 0 {
		in.Type = ApplicationCommandTypeChatInput
	}
	payloadJSON := utils.Must1(json.Marshal(in))

	path := fmt.Sprintf("/applications/%s/guilds/%s/commands", config.Config.Discord.BotUserID, config.Config.Discord.GuildID)
	return discard(doSimplePostishRequest(ctx,
		"Create Guild Application Command",
		http.MethodPost, path,
		"application/json", payloadJSON,
	))
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#create-interaction-response
func CreateInteractionResponse(ctx context.Context, interactionID, interactionToken string, in InteractionResponse) error {
	payloadJSON, err := json.Marshal(in)
	if err != nil {
		return oops.New(nil, "failed to marshal request body")
	}

	// NOTE(ben): There is a flavor of this API now where we could get an
	// Interaction Callback Response. Evidently we don't need this right now, but
	// in the future it might be good to switch to that version here.
	path := fmt.Sprintf("/interactions/%s/%s/callback", interactionID, interactionToken)
	return discard(doSimplePostishRequest(ctx,
		"Create Interaction Response",
		http.MethodPost, path,
		"application/json", payloadJSON,
	))
}

// https://docs.discord.com/developers/interactions/receiving-and-responding#edit-original-interaction-response
func EditOriginalInteractionResponse(ctx context.Context, interactionToken string, payloadJSON string, files ...FileUpload) (Message, error) {
	contentType, body := makeNewMessageBody(payloadJSON, files)

	path := fmt.Sprintf("/webhooks/%s/%s/messages/@original", config.Config.Discord.BotUserID, interactionToken)
	return unmarshal[Message](doSimplePostishRequest(ctx,
		"Edit Original Interaction Response",
		http.MethodPatch, path,
		contentType, body,
	))
}

func GetAuthorizeUrl(state string, includeEmail bool) string {
	scope := "identify"
	if includeEmail {
		scope = "identify email"
	}

	params := make(url.Values)
	params.Set("response_type", "code")
	params.Set("client_id", config.Config.Discord.OAuthClientID)
	params.Set("scope", scope)
	params.Set("prompt", "none") // immediately redirect back to HMN if already authorized
	params.Set("state", state)
	params.Set("redirect_uri", hmnurl.BuildDiscordOAuthCallback())
	return fmt.Sprintf("%s?%s", buildUrl("/oauth2/authorize"), params.Encode())
}

type FileUpload struct {
	Name string
	Data []byte
}

func makeNewMessageBody(payloadJSON string, files []FileUpload) (contentType string, body []byte) {
	if len(files) == 0 {
		contentType = "application/json"
		body = []byte(payloadJSON)
	} else {
		var bodyBuffer bytes.Buffer
		w := multipart.NewWriter(&bodyBuffer)
		contentType = w.FormDataContentType()

		jsonHeader := textproto.MIMEHeader{}
		jsonHeader.Set("Content-Disposition", `form-data; name="payload_json"`)
		jsonHeader.Set("Content-Type", "application/json")
		jsonWriter, _ := w.CreatePart(jsonHeader)
		jsonWriter.Write([]byte(payloadJSON))

		for _, f := range files {
			formFile, _ := w.CreateFormFile("file", f.Name)
			formFile.Write(f.Data)
		}

		w.Close()

		body = bodyBuffer.Bytes()
	}

	if len(body) == 0 {
		panic("somehow we generated an empty body for Discord")
	}

	return
}

func logErrorResponse(ctx context.Context, res *http.Response, bodyBytes []byte, msg string) {
	logging.ExtractLogger(ctx).Error().Msg(msg)
	res.Write(os.Stderr)
	os.Stderr.Write(bodyBytes)
}

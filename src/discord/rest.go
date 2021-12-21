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
	"net/http/httputil"
	"net/textproto"
	"net/url"
	"strconv"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
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

func makeRequest(ctx context.Context, method string, path string, body []byte) *http.Request {
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

type GetGatewayBotResponse struct {
	URL string `json:"url"`
	// We don't care about shards or session limit stuff; we will never hit those limits
}

func GetGatewayBot(ctx context.Context) (*GetGatewayBotResponse, error) {
	const name = "Get Gateway Bot"

	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		return makeRequest(ctx, http.MethodGet, "/gateway/bot", nil)
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != 200 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var result GetGatewayBotResponse
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord response")
	}

	return &result, nil
}

func GetGuildRoles(ctx context.Context, guildID string) ([]Role, error) {
	const name = "Get Guild Roles"

	path := fmt.Sprintf("/guilds/%s/roles", guildID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		return makeRequest(ctx, http.MethodGet, path, nil)
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var roles []Role
	err = json.Unmarshal(bodyBytes, &roles)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord message")
	}

	return roles, nil
}

func GetGuildChannels(ctx context.Context, guildID string) ([]Channel, error) {
	const name = "Get Guild Channels"

	path := fmt.Sprintf("/guilds/%s/channels", guildID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		return makeRequest(ctx, http.MethodGet, path, nil)
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var channels []Channel
	err = json.Unmarshal(bodyBytes, &channels)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord message")
	}

	return channels, nil
}

func GetGuildMember(ctx context.Context, guildID, userID string) (*GuildMember, error) {
	const name = "Get Guild Member"

	path := fmt.Sprintf("/guilds/%s/members/%s", guildID, userID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		return makeRequest(ctx, http.MethodGet, path, nil)
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, NotFound
	} else if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var msg GuildMember
	err = json.Unmarshal(bodyBytes, &msg)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord message")
	}

	return &msg, nil
}

type CreateMessageRequest struct {
	Content string `json:"content"`
}

func CreateMessage(ctx context.Context, channelID string, payloadJSON string, files ...FileUpload) (*Message, error) {
	const name = "Create Message"

	contentType, body := makeNewMessageBody(payloadJSON, files)

	path := fmt.Sprintf("/channels/%s/messages", channelID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req := makeRequest(ctx, http.MethodPost, path, body)
		req.Header.Add("Content-Type", contentType)
		return req
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	// Maybe in the future we could more nicely handle errors like "bad channel",
	// but honestly what are the odds that we mess that up...

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var msg Message
	err = json.Unmarshal(bodyBytes, &msg)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord message")
	}

	return &msg, nil
}

func DeleteMessage(ctx context.Context, channelID string, messageID string) error {
	const name = "Delete Message"

	path := fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		return makeRequest(ctx, http.MethodDelete, path, nil)
	})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		logErrorResponse(ctx, name, res, "")
		return oops.New(nil, "got unexpected status code when deleting message")
	}

	return nil
}

func CreateDM(ctx context.Context, recipientID string) (*Channel, error) {
	const name = "Create DM"

	path := "/users/@me/channels"
	body := []byte(fmt.Sprintf(`{"recipient_id":"%s"}`, recipientID))
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req := makeRequest(ctx, http.MethodPost, path, body)
		req.Header.Add("Content-Type", "application/json")
		return req
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var channel Channel
	err = json.Unmarshal(bodyBytes, &channel)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord channel")
	}

	return &channel, nil
}

type OAuthCodeExchangeResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

func ExchangeOAuthCode(ctx context.Context, code, redirectURI string) (*OAuthCodeExchangeResponse, error) {
	const name = "OAuth Code Exchange"

	body := make(url.Values)
	body.Set("client_id", config.Config.Discord.OAuthClientID)
	body.Set("client_secret", config.Config.Discord.OAuthClientSecret)
	body.Set("grant_type", "authorization_code")
	body.Set("code", code)
	body.Set("redirect_uri", redirectURI)
	bodyStr := body.Encode()

	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req := makeRequest(ctx, http.MethodPost, "/oauth2/token", []byte(bodyStr))
		req.Header.Add("Content-Type", "application/x-www-form-urlencoded")

		return req
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var tokenResponse OAuthCodeExchangeResponse
	err = json.Unmarshal(bodyBytes, &tokenResponse)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord OAuth token")
	}

	return &tokenResponse, nil
}

func GetCurrentUserAsOAuth(ctx context.Context, accessToken string) (*User, error) {
	const name = "Get Current User"

	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, buildUrl("/users/@me"), nil)
		if err != nil {
			panic(err)
		}
		req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", accessToken))
		req.Header.Add("User-Agent", UserAgent)

		return req
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var user User
	err = json.Unmarshal(bodyBytes, &user)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord user")
	}

	return &user, nil
}

func AddGuildMemberRole(ctx context.Context, userID, roleID string) error {
	const name = "Add Guild Member Role"

	path := fmt.Sprintf("/guilds/%s/members/%s/roles/%s", config.Config.Discord.GuildID, userID, roleID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		return makeRequest(ctx, http.MethodPut, path, nil)
	})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		logErrorResponse(ctx, name, res, "")
		return oops.New(nil, "got unexpected status code when adding role")
	}

	return nil
}

func RemoveGuildMemberRole(ctx context.Context, userID, roleID string) error {
	const name = "Remove Guild Member Role"

	path := fmt.Sprintf("/guilds/%s/members/%s/roles/%s", config.Config.Discord.GuildID, userID, roleID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		return makeRequest(ctx, http.MethodDelete, path, nil)
	})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusNoContent {
		logErrorResponse(ctx, name, res, "")
		return oops.New(nil, "got unexpected status code when removing role")
	}

	return nil
}

func GetChannelMessage(ctx context.Context, channelID, messageID string) (*Message, error) {
	const name = "Get Channel Message"

	path := fmt.Sprintf("/channels/%s/messages/%s", channelID, messageID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		return makeRequest(ctx, http.MethodGet, path, nil)
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode == http.StatusNotFound {
		return nil, NotFound
	} else if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var msg Message
	err = json.Unmarshal(bodyBytes, &msg)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord message")
	}

	return &msg, nil
}

type GetChannelMessagesInput struct {
	Around string
	Before string
	After  string
	Limit  int
}

func GetChannelMessages(ctx context.Context, channelID string, in GetChannelMessagesInput) ([]Message, error) {
	const name = "Get Channel Messages"

	path := fmt.Sprintf("/channels/%s/messages", channelID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req := makeRequest(ctx, http.MethodGet, path, nil)
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
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var msgs []Message
	err = json.Unmarshal(bodyBytes, &msgs)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord message")
	}

	return msgs, nil
}

// See https://discord.com/developers/docs/interactions/application-commands#create-guild-application-command-json-params
type CreateGuildApplicationCommandRequest struct {
	Name              string                     `json:"name"`               // 1-32 character name
	Description       string                     `json:"description"`        // 1-100 character description
	Options           []ApplicationCommandOption `json:"options"`            // the parameters for the command
	DefaultPermission *bool                      `json:"default_permission"` // whether the command is enabled by default when the app is added to a guild
	Type              ApplicationCommandType     `json:"type"`               // the type of command, defaults 1 if not set
}

// See https://discord.com/developers/docs/interactions/application-commands#create-guild-application-command
func CreateGuildApplicationCommand(ctx context.Context, in CreateGuildApplicationCommandRequest) error {
	const name = "Create Guild Application Command"

	if in.Type == 0 {
		in.Type = ApplicationCommandTypeChatInput
	}

	payloadJSON, err := json.Marshal(in)
	if err != nil {
		return oops.New(nil, "failed to marshal request body")
	}

	path := fmt.Sprintf("/applications/%s/guilds/%s/commands", config.Config.Discord.BotUserID, config.Config.Discord.GuildID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req := makeRequest(ctx, http.MethodPost, path, []byte(payloadJSON))
		req.Header.Add("Content-Type", "application/json")
		return req
	})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return oops.New(nil, "received error from Discord")
	}

	return nil
}

func CreateInteractionResponse(ctx context.Context, interactionID, interactionToken string, in InteractionResponse) error {
	const name = "Create Interaction Response"

	payloadJSON, err := json.Marshal(in)
	if err != nil {
		return oops.New(nil, "failed to marshal request body")
	}

	path := fmt.Sprintf("/interactions/%s/%s/callback", interactionID, interactionToken)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req := makeRequest(ctx, http.MethodPost, path, []byte(payloadJSON))
		req.Header.Add("Content-Type", "application/json")
		return req
	})
	if err != nil {
		return err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return oops.New(nil, "received error from Discord")
	}

	return nil
}

func EditOriginalInteractionResponse(ctx context.Context, interactionToken string, payloadJSON string, files ...FileUpload) (*Message, error) {
	const name = "Edit Original Interaction Response"

	contentType, body := makeNewMessageBody(payloadJSON, files)

	path := fmt.Sprintf("/webhooks/%s/%s/messages/@original", config.Config.Discord.BotUserID, interactionToken)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req := makeRequest(ctx, http.MethodPatch, path, body)
		req.Header.Add("Content-Type", contentType)
		return req
	})
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode >= 400 {
		logErrorResponse(ctx, name, res, "")
		return nil, oops.New(nil, "received error from Discord")
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		panic(err)
	}

	var msg Message
	err = json.Unmarshal(bodyBytes, &msg)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord message")
	}

	return &msg, nil
}

func GetAuthorizeUrl(state string) string {
	params := make(url.Values)
	params.Set("response_type", "code")
	params.Set("client_id", config.Config.Discord.OAuthClientID)
	params.Set("scope", "identify")
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

func logErrorResponse(ctx context.Context, name string, res *http.Response, msg string) {
	dump, err := httputil.DumpResponse(res, true)
	if err != nil {
		panic(err)
	}

	logging.ExtractLogger(ctx).Error().Str("name", name).Msg(msg)
	fmt.Println(string(dump))
}

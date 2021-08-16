package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"

	"git.handmade.network/hmn/hmn/src/config"
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

type CreateMessageRequest struct {
	Content string `json:"content"`
}

func CreateMessage(ctx context.Context, channelID string, payloadJSON string) (*Message, error) {
	const name = "Create Message"

	path := fmt.Sprintf("/channels/%s/messages", channelID)
	res, err := doWithRateLimiting(ctx, name, func(ctx context.Context) *http.Request {
		req := makeRequest(ctx, http.MethodPost, path, []byte(payloadJSON))
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
		req, err := http.NewRequestWithContext(
			ctx,
			http.MethodPost,
			"https://discord.com/api/oauth2/token",
			strings.NewReader(bodyStr),
		)
		if err != nil {
			panic(err)
		}
		req.Header.Add("User-Agent", UserAgent)
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
	const name = "Delete Message"

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

func logErrorResponse(ctx context.Context, name string, res *http.Response, msg string) {
	dump, err := httputil.DumpResponse(res, true)
	if err != nil {
		panic(err)
	}

	logging.ExtractLogger(ctx).Error().Str("name", name).Msg(msg)
	fmt.Println(string(dump))
}

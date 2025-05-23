package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/perf"
	"git.handmade.network/hmn/hmn/src/utils"
)

var twitchAPIBaseUrl = config.Config.Twitch.BaseUrl

var HitRateLimit = errors.New("hit rate limit")
var MaxRetries = errors.New("hit max retries")

var httpClient = &http.Client{}

// NOTE(asaf): Access token is not thread-safe right now.
//
//	All twitch requests are made through the goroutine in MonitorTwitchSubscriptions.
var activeAccessToken string
var rateLimitReset time.Time

type twitchUser struct {
	TwitchID    string
	TwitchLogin string
}

func getTwitchUsersByLogin(ctx context.Context, logins []string) ([]twitchUser, error) {
	defer perf.StartBlock(ctx, "TwitchAPI", "Fetch twitch user info").End()

	result := make([]twitchUser, 0, len(logins))
	numChunks := len(logins)/100 + 1
	for i := 0; i < numChunks; i++ {
		query := url.Values{}
		query.Add("first", "100")
		for _, login := range logins[i*100 : utils.Min((i+1)*100, len(logins))] {
			query.Add("login", login)
		}
		req, err := http.NewRequestWithContext(ctx, "GET", buildUrl("/users", query.Encode()), nil)
		if err != nil {
			return nil, oops.New(err, "failed to create requset")
		}
		res, err := doRequest(ctx, true, req)
		if err != nil {
			return nil, oops.New(err, "failed to fetch twitch users")
		}

		type user struct {
			ID    string `json:"id"`
			Login string `json:"login"`
		}

		type twitchResponse struct {
			Data []user `json:"data"`
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, oops.New(err, "failed to read response body while fetching twitch users")
		}

		var userResponse twitchResponse
		err = json.Unmarshal(body, &userResponse)
		if err != nil {
			return nil, oops.New(err, "failed to parse twitch response while fetching twitch users")
		}

		for _, u := range userResponse.Data {
			result = append(result, twitchUser{
				TwitchID:    u.ID,
				TwitchLogin: u.Login,
			})
		}
	}

	return result, nil
}

func getStreamStatus(ctx context.Context, twitchIDs []string) ([]streamStatus, error) {
	defer perf.StartBlock(ctx, "TwitchAPI", "Fetch stream status").End()

	result := make([]streamStatus, 0, len(twitchIDs))
	numChunks := len(twitchIDs)/100 + 1
	for i := 0; i < numChunks; i++ {
		query := url.Values{}
		query.Add("first", "100")
		for _, tid := range twitchIDs[i*100 : utils.Min((i+1)*100, len(twitchIDs))] {
			query.Add("user_id", tid)
		}
		req, err := http.NewRequestWithContext(ctx, "GET", buildUrl("/streams", query.Encode()), nil)
		if err != nil {
			return nil, oops.New(err, "failed to create request")
		}
		res, err := doRequest(ctx, true, req)
		if err != nil {
			return nil, oops.New(err, "failed to fetch stream statuses")
		}

		type twitchStatus struct {
			StreamID    string   `json:"id"`
			TwitchID    string   `json:"user_id"`
			TwitchLogin string   `json:"user_login"`
			GameID      string   `json:"game_id"`
			Type        string   `json:"type"`
			Title       string   `json:"title"`
			StartedAt   string   `json:"started_at"`
			Thumbnail   string   `json:"thumbnail_url"`
			Tags        []string `json:"tags"`
		}

		type twitchResponse struct {
			Data []twitchStatus `json:"data"`
		}
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, oops.New(err, "failed to read response body while processing stream statuses")
		}
		log := logging.ExtractLogger(ctx)
		log.Debug().Str("getStreamStatus response", string(body)).Msg("Got getStreamStatus response")

		var streamResponse twitchResponse
		err = json.Unmarshal(body, &streamResponse)
		if err != nil {
			return nil, oops.New(err, "failed to parse twitch response while processing stream statuses")
		}

		for _, d := range streamResponse.Data {
			started, err := time.Parse(time.RFC3339, d.StartedAt)
			if err != nil {
				logging.ExtractLogger(ctx).Warn().Str("Time string", d.StartedAt).Msg("Failed to parse twitch timestamp")
				started = time.Now()
			}
			status := streamStatus{
				StreamID:    d.StreamID,
				TwitchID:    d.TwitchID,
				TwitchLogin: d.TwitchLogin,
				Live:        d.Type == "live",
				Title:       d.Title,
				StartedAt:   started,
				CategoryID:  d.GameID,
				Tags:        d.Tags,
			}
			result = append(result, status)
		}
	}

	return result, nil
}

type archivedVideo struct {
	ID           string
	StreamID     string
	TwitchID     string
	TwitchLogin  string
	Title        string
	Description  string
	CreatedAt    time.Time
	Duration     time.Duration
	VODUrl       string
	VODThumbnail string
	RawDuration  string
	RawCreatedAt string
}

func getArchivedVideosForUser(ctx context.Context, twitchID string, numVODs int) ([]archivedVideo, error) {
	query := url.Values{}
	query.Add("user_id", twitchID)
	query.Add("type", "archive")
	query.Add("first", strconv.Itoa(numVODs))

	return getArchivedVideosByQuery(ctx, query)
}

func getArchivedVideos(ctx context.Context, videoIDs []string) ([]archivedVideo, error) {
	query := url.Values{}
	for _, vid := range videoIDs {
		query.Add("id", vid)
	}
	return getArchivedVideosByQuery(ctx, query)
}

func getArchivedVideosByQuery(ctx context.Context, query url.Values) ([]archivedVideo, error) {
	defer perf.StartBlock(ctx, "TwitchAPI", "Query archived videos").End()

	req, err := http.NewRequestWithContext(ctx, "GET", buildUrl("/videos", query.Encode()), nil)
	if err != nil {
		return nil, oops.New(err, "failed to create request")
	}
	res, err := doRequest(ctx, true, req)
	if err != nil {
		return nil, oops.New(err, "failed to fetch archived videos for user")
	}

	type twitchVideo struct {
		ID           string `json:"id"`
		StreamID     string `json:"stream_id"`
		UserID       string `json:"user_id"`
		UserLogin    string `json:"user_login"`
		UserName     string `json:"user_name"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		CreatedAt    string `json:"created_at"`
		PublishedAt  string `json:"published_at"`
		Url          string `json:"url"`
		ThumbnailUrl string `json:"thumbnail_url"`
		Viewable     string `json:"viewable"`
		ViewCount    int    `json:"view_count"`
		Language     string `json:"language"`
		Type         string `json:"type"`
		Duration     string `json:"duration"`
	}

	type twitchResponse struct {
		Data []twitchVideo `json:"data"`
	}
	body, err := io.ReadAll(res.Body)
	res.Body.Close()
	if err != nil {
		return nil, oops.New(err, "failed to read response body while processing archived videos")
	}
	log := logging.ExtractLogger(ctx)
	log.Debug().Str("getArchivedVideosForUser response", string(body)).Msg("Got getArchivedVideosForUser response")

	var resp twitchResponse
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return nil, oops.New(err, "failed to parse twitch response while processing archived videos")
	}

	var result []archivedVideo

	for _, v := range resp.Data {
		createdAt, err := time.Parse(time.RFC3339, v.CreatedAt)
		if err != nil {
			logging.ExtractLogger(ctx).Warn().Str("Time string", v.CreatedAt).Msg("Failed to parse twitch timestamp")
			createdAt = time.Time{}
		}
		duration, err := time.ParseDuration(v.Duration)
		if err != nil {
			duration = 0
		}
		archived := archivedVideo{
			ID:           v.ID,
			StreamID:     v.StreamID,
			TwitchID:     v.UserID,
			TwitchLogin:  v.UserLogin,
			Title:        v.Title,
			Description:  v.Description,
			CreatedAt:    createdAt,
			Duration:     duration,
			VODUrl:       v.Url,
			VODThumbnail: v.ThumbnailUrl,
			RawDuration:  v.Duration,
			RawCreatedAt: v.CreatedAt,
		}
		result = append(result, archived)
	}
	return result, nil
}

type twitchEventSub struct {
	EventID    string
	TwitchID   string
	Type       string
	GoodStatus bool
}

func getEventSubscriptions(ctx context.Context) ([]twitchEventSub, error) {
	defer perf.StartBlock(ctx, "TwitchAPI", "Fetch event subscriptions").End()

	result := make([]twitchEventSub, 0)
	after := ""
	for {
		query := url.Values{}
		if len(after) > 0 {
			query.Add("after", after)
		}
		req, err := http.NewRequestWithContext(ctx, "GET", buildUrl("/eventsub/subscriptions", query.Encode()), nil)
		if err != nil {
			return nil, oops.New(err, "failed to create request")
		}
		res, err := doRequest(ctx, true, req)
		if err != nil {
			return nil, oops.New(err, "failed to fetch twitch event subscriptions")
		}

		type eventSub struct {
			ID        string `json:"id"`
			Status    string `json:"status"`
			Type      string `json:"type"`
			Condition struct {
				TwitchID string `json:"broadcaster_user_id"`
			} `json:"condition"`
		}

		type twitchResponse struct {
			Data       []eventSub `json:"data"`
			Pagination *struct {
				After string `json:"cursor"`
			} `json:"pagination"`
		}

		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, oops.New(err, "failed to read response body while fetching twitch eventsubs")
		}

		var eventSubResponse twitchResponse
		err = json.Unmarshal(body, &eventSubResponse)
		if err != nil {
			return nil, oops.New(err, "failed to parse twitch response while fetching twitch eventsubs")
		}

		for _, es := range eventSubResponse.Data {
			result = append(result, twitchEventSub{
				EventID:    es.ID,
				TwitchID:   es.Condition.TwitchID,
				Type:       es.Type,
				GoodStatus: es.Status == "enabled" || es.Status == "webhook_callback_verification_pending",
			})
		}

		if eventSubResponse.Pagination == nil || eventSubResponse.Pagination.After == "" {
			return result, nil
		} else {
			after = eventSubResponse.Pagination.After
		}
	}
}

func subscribeToEvent(ctx context.Context, eventType string, twitchID string) error {
	defer perf.StartBlock(ctx, "TwitchAPI", fmt.Sprintf("Subscribe to event: %s", eventType)).End()

	type eventBody struct {
		Type      string `json:"type"`
		Version   string `json:"version"`
		Condition struct {
			TwitchID string `json:"broadcaster_user_id"`
		} `json:"condition"`
		Transport struct {
			Method   string `json:"method"`
			Callback string `json:"callback"`
			Secret   string `json:"secret"`
		} `json:"transport"`
	}

	ev := eventBody{
		Type:    eventType,
		Version: "1",
	}
	ev.Condition.TwitchID = twitchID
	ev.Transport.Method = "webhook"
	// NOTE(asaf): Twitch has special treatment for localhost. We can keep this around for live/beta because it just won't replace anything.
	ev.Transport.Callback = strings.ReplaceAll(hmnurl.BuildTwitchEventSubCallback(), "handmade.local:9001", "localhost")
	ev.Transport.Secret = config.Config.Twitch.EventSubSecret

	evJson, err := json.Marshal(ev)
	if err != nil {
		return oops.New(err, "failed to marshal event sub data")
	}
	req, err := http.NewRequestWithContext(ctx, "POST", buildUrl("/eventsub/subscriptions", ""), bytes.NewReader(evJson))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		return oops.New(err, "failed to create request")
	}
	res, err := doRequest(ctx, true, req)
	if err != nil {
		return oops.New(err, "failed to create new event subscription")
	}
	defer readAndClose(res)

	if res.StatusCode >= 300 {
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return oops.New(err, "failed to read response body while creating twitch eventsubs")
		}
		logging.ExtractLogger(ctx).Error().Interface("Headers", res.Header).Int("Status code", res.StatusCode).Str("Body", string(body[:])).Msg("Failed to create twitch event sub")
		return oops.New(nil, "failed to create new event subscription")
	}
	return nil
}

func unsubscribeFromEvent(ctx context.Context, eventID string) error {
	defer perf.StartBlock(ctx, "TwitchAPI", "Unsubscribe from event").End()

	query := url.Values{}
	query.Add("id", eventID)
	req, err := http.NewRequestWithContext(ctx, "DELETE", buildUrl("/eventsub/subscriptions", query.Encode()), nil)
	if err != nil {
		return oops.New(err, "failed to create request")
	}
	res, err := doRequest(ctx, true, req)
	if err != nil {
		return oops.New(err, "failed to delete new event subscription")
	}
	defer readAndClose(res)

	if res.StatusCode > 300 {
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return oops.New(err, "failed to read response body while deleting twitch eventsubs")
		}
		logging.ExtractLogger(ctx).Error().Interface("Headers", res.Header).Int("Status code", res.StatusCode).Str("Body", string(body[:])).Msg("Failed to delete twitch event sub")
		return oops.New(nil, "failed to delete new event subscription")
	}
	return nil
}

var twitchNotFound = errors.New("Twitch API 404")

func doRequest(ctx context.Context, waitOnRateLimit bool, req *http.Request) (*http.Response, error) {
	defer perf.StartBlock(ctx, "TwitchAPI", "API Request").End()

	serviceUnavailable := false
	numRetries := 5

	for {
		if numRetries == 0 {
			return nil, MaxRetries
		}
		numRetries -= 1

		now := time.Now()
		if rateLimitReset.After(now) {
			if waitOnRateLimit {
				timer := time.NewTimer(rateLimitReset.Sub(now))
				select {
				case <-timer.C:
				case <-ctx.Done():
					return nil, errors.New("request interrupted during rate limiting")
				}
			} else {
				return nil, HitRateLimit
			}
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", activeAccessToken))
		req.Header.Set("Client-Id", config.Config.Twitch.ClientID)
		res, err := httpClient.Do(req)
		if err != nil {
			return nil, oops.New(err, "twitch request failed")
		}

		if res.StatusCode != 503 {
			serviceUnavailable = false
		}

		if res.StatusCode >= 200 && res.StatusCode < 300 {
			return res, nil
		} else if res.StatusCode == 503 {
			readAndClose(res)
			if serviceUnavailable {
				// NOTE(asaf): The docs say we should retry once if we receive 503
				return nil, oops.New(nil, "got 503 Service Unavailable twice in a row")
			} else {
				serviceUnavailable = true
			}
		} else if res.StatusCode == 429 {
			logging.ExtractLogger(ctx).Warn().Interface("Headers", res.Header).Msg("Hit Twitch rate limit")
			err = updateRateLimitReset(res)
			if err != nil {
				return nil, err
			}
		} else if res.StatusCode == 401 {
			logging.ExtractLogger(ctx).Warn().Msg("Twitch refresh token is invalid. Renewing...")
			readAndClose(res)
			err = refreshAccessToken(ctx)
			if err != nil {
				return nil, err
			}
		} else if res.StatusCode == 404 {
			return nil, twitchNotFound
		} else {
			body, err := io.ReadAll(res.Body)
			if err != nil {
				return nil, oops.New(err, "failed to read response body")
			}
			logging.ExtractLogger(ctx).Error().Interface("Headers", res.Header).Int("Status code", res.StatusCode).Str("Body", string(body[:])).Msg("Unexpected status code from twitch")
			res.Body.Close()
			return res, oops.New(nil, "got an unexpected status code from twitch")
		}
	}
}

func updateRateLimitReset(res *http.Response) error {
	defer readAndClose(res)

	resetStr := res.Header.Get("Ratelimit-Reset")
	if len(resetStr) == 0 {
		return oops.New(nil, "no ratelimit data on response")
	}

	resetUnix, err := strconv.Atoi(resetStr)
	if err != nil {
		return oops.New(err, "failed to parse reset time")
	}

	rateLimitReset = time.Unix(int64(resetUnix), 0)
	return nil
}

type AccessTokenResponse struct {
	AccessToken string `json:"access_token"`
}

func refreshAccessToken(ctx context.Context) error {
	defer perf.StartBlock(ctx, "TwitchAPI", "Refresh access token").End()

	logging.ExtractLogger(ctx).Info().Msg("Refreshing twitch token")
	query := url.Values{}
	query.Add("client_id", config.Config.Twitch.ClientID)
	query.Add("client_secret", config.Config.Twitch.ClientSecret)
	query.Add("grant_type", "client_credentials")
	url := fmt.Sprintf("%s/token?%s", config.Config.Twitch.BaseIDUrl, query.Encode())
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return oops.New(err, "failed to create request")
	}

	res, err := httpClient.Do(req)
	if err != nil {
		return oops.New(err, "failed to request new access token")
	}
	defer readAndClose(res)

	if res.StatusCode >= 400 {
		// NOTE(asaf): The docs don't specify the error cases for this call.
		// NOTE(asaf): According to the docs rate limiting is per-token, and we don't use a token for this call,
		//             so who knows how rate limiting works here.
		body, _ := io.ReadAll(res.Body)
		logging.ExtractLogger(ctx).Error().Interface("Headers", res.Header).Int("Status code", res.StatusCode).Str("body", string(body[:])).Msg("Got bad status code from twitch access token refresh")
		return oops.New(nil, "received unexpected status code from twitch access token refresh")
	}

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return oops.New(err, "failed to read response body")
	}
	var accessTokenResponse AccessTokenResponse
	err = json.Unmarshal(body, &accessTokenResponse)
	if err != nil {
		return oops.New(err, "failed to unmarshal access token response")
	}

	activeAccessToken = accessTokenResponse.AccessToken
	return nil
}

func readAndClose(res *http.Response) {
	io.ReadAll(res.Body)
	res.Body.Close()
}

func buildUrl(path string, queryParams string) string {
	return fmt.Sprintf("%s%s?%s", config.Config.Twitch.BaseUrl, path, queryParams)
}

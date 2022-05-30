package website

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/twitch"
)

func TwitchEventSubCallback(c *RequestContext) ResponseData {
	secret := config.Config.Twitch.EventSubSecret
	messageId := c.Req.Header.Get("Twitch-Eventsub-Message-Id")
	timestamp := c.Req.Header.Get("Twitch-Eventsub-Message-Timestamp")
	signature := c.Req.Header.Get("Twitch-Eventsub-Message-Signature")
	messageType := c.Req.Header.Get("Twitch-Eventsub-Message-Type")

	body, err := io.ReadAll(c.Req.Body)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to read request body"))
	}

	hmacMessage := fmt.Sprintf("%s%s%s", messageId, timestamp, string(body[:]))
	hmac := hmac.New(sha256.New, []byte(secret))
	hmac.Write([]byte(hmacMessage))
	hash := hmac.Sum(nil)
	hmacStr := "sha256=" + hex.EncodeToString(hash)

	if hmacStr != signature {
		var res ResponseData
		res.StatusCode = 403
		return res
	}

	c.Logger.Debug().Str("Body", string(body[:])).Str("Type", messageType).Msg("Got twitch webhook")

	if messageType == "webhook_callback_verification" {
		type challengeReq struct {
			Challenge string `json:"challenge"`
		}
		var data challengeReq
		err = json.Unmarshal(body, &data)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to unmarshal twitch verification"))
		}
		var res ResponseData
		res.StatusCode = 200

		res.Header().Set("Content-Type", "text/plain") // NOTE(asaf): No idea why, but the twitch-cli fails when we don't set this.
		res.Write([]byte(data.Challenge))
		return res
	} else {
		err := twitch.QueueTwitchNotification(messageType, body)
		if err != nil {
			c.Logger.Error().Err(err).Msg("Failed to process twitch callback")
			// NOTE(asaf): Returning 200 either way here
		}
		var res ResponseData
		res.StatusCode = 200
		return res
	}
}

func TwitchDebugPage(c *RequestContext) ResponseData {
	streams, err := db.Query[models.TwitchStream](c.Context(), c.Conn,
		`
		SELECT $columns
		FROM
			twitch_stream
		ORDER BY started_at DESC
		`,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch twitch streams"))
	}

	html := ""
	for _, s := range streams {
		html += fmt.Sprintf(`<a href="https://twitch.tv/%s">%s</a>%s<br />`, s.Login, s.Login, s.Title)
	}
	var res ResponseData
	res.StatusCode = 200
	res.Write([]byte(html))
	return res
}

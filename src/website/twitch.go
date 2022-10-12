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
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
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
		err := twitch.QueueTwitchNotification(c, c.Conn, messageType, body)
		if err != nil {
			c.Logger.Error().Err(err).Msg("Failed to process twitch callback")
			// NOTE(asaf): Returning 200 either way here
		}
		var res ResponseData
		res.StatusCode = 200
		return res
	}
}

type TwitchDebugData struct {
	templates.BaseData
	DataJson string
}

func TwitchDebugPage(c *RequestContext) ResponseData {
	type dataUser struct {
		Login string `json:"login"`
		Live  bool   `json:"live"`
	}
	type dataLog struct {
		ID       int    `json:"id"`
		LoggedAt int64  `json:"loggedAt"`
		Type     string `json:"type"`
		Login    string `json:"login"`
		Message  string `json:"message"`
		Payload  string `json:"payload"`
	}
	type dataJson struct {
		Users []dataUser `json:"users"`
		Logs  []dataLog  `json:"logs"`
	}
	streamers, err := hmndata.FetchTwitchStreamers(c, c.Conn)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch twitch streamers"))
	}
	live, err := db.Query[models.TwitchLatestStatus](c, c.Conn,
		`
		SELECT $columns
		FROM
			twitch_latest_status
		WHERE
			live = TRUE
		`,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch live twitch streamers"))
	}
	logs, err := db.Query[models.TwitchLog](c, c.Conn,
		`
		SELECT $columns
		FROM twitch_log
		ORDER BY logged_at DESC, id DESC
		`,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch twitch logs"))
	}

	var data dataJson
	for _, u := range streamers {
		var user dataUser
		user.Login = u.TwitchLogin
		user.Live = false
		for _, l := range live {
			if l.TwitchLogin == u.TwitchLogin {
				user.Live = true
				break
			}
		}
		data.Users = append(data.Users, user)
	}
	messageTypes := []string{
		"",
		"Other",
		"Hook",
		"REST",
	}
	data.Logs = make([]dataLog, 0, 0)
	for _, l := range logs {
		var log dataLog
		log.ID = l.ID
		log.LoggedAt = l.LoggedAt.UnixMilli()
		log.Login = l.Login
		log.Type = messageTypes[l.Type]
		log.Message = l.Message
		log.Payload = l.Payload
		data.Logs = append(data.Logs, log)
	}
	jsonStr, err := json.Marshal(data)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to stringify twitch logs"))
	}
	var res ResponseData
	res.MustWriteTemplate("twitch_debug.html", TwitchDebugData{
		BaseData: getBaseDataAutocrumb(c, "Twitch Debug"),
		DataJson: string(jsonStr),
	}, c.Perf)
	return res
}

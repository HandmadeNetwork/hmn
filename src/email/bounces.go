package email

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

var badCodeRegex = regexp.MustCompile(`^(smtp;)?((421)|(450)|(451)|(452)|(552))`)

var httpClient = &http.Client{}

type Bounce struct {
	// RecordType    string `json:"RecordType"`
	// ID            string `json:"ID"`
	// Type          string `json:"Type"`
	// TypeCode      int    `json:"TypeCode"`
	// Name          string `json:"Name"`
	// Tag           string `json:"Tag"`
	// MessageID     string `json:"MessageID"`
	// ServerID      string `json:"ServerID"`
	// MessageStream string `json:"MessageStream"`
	// Description   string `json:"Description"`
	Details   string `json:"Details"`
	Email     string `json:"Email"`
	From      string `json:"From"`
	BouncedAt string `json:"BouncedAt"`
	// DumpAvailable bool   `json:"DumpAvailable"`
	// Inactive      bool   `json:"Inactive"`
	// CanActivate   bool   `json:"CanActivate"`
	Subject string `json:"Subject"`
	// Content       string `json:"Content"`
}

func MonitorBounces(ctx context.Context, conn *pgxpool.Pool) jobs.Job {
	log := logging.ExtractLogger(ctx).With().Str("email goroutine", "bounce monitoring").Logger()
	ctx = logging.AttachLoggerToContext(&log, ctx)

	if config.Config.Postmark.TransactionalStreamToken == "" {
		log.Warn().Msg("No postmark token provided.")
		return jobs.Noop()
	}

	job := jobs.New()

	go func() {
		defer func() {
			log.Info().Msg("Shutting down email bounce monitor")
			job.Done()
		}()
		log.Info().Msg("Running email bounce monitor...")

		monitorTimer := utils.MakeAutoResetTimer(ctx, 30*time.Minute, true)

		for {
			done, err := func() (done bool, retErr error) {
				defer utils.RecoverPanicAsError(&retErr)

				select {
				case <-ctx.Done():
					return true, nil
				case <-monitorTimer.C:
					lastBounceDate, err := db.QueryOneScalar[time.Time](ctx, conn,
						`
						SELECT bounced_at
						FROM email_blacklist
						ORDER BY bounced_at DESC
						LIMIT 1
						`,
					)
					if err != nil && !errors.Is(err, db.NotFound) {
						log.Error().Err(err).Msg("Failed to query latest blacklist date")
						return
					}

					if lastBounceDate.IsZero() {
						lastBounceDate = time.Now().Add(-45 * (time.Hour * 24))
					}
					logging.Debug().Interface("last", lastBounceDate).Msg("Fetching bounces from postmark")
					bounces, err := PostmarkGetBounces(ctx, lastBounceDate)
					if err != nil {
						log.Error().Err(err).Msg("Error while requesting bounces from postmark")
						return
					}

					logging.Debug().Int("Num bounces", len(bounces)).Msg("Got bounces")
					newBlacklists := make([]models.EmailBlacklist, 0, len(bounces))
					now := time.Now()
					for _, b := range bounces {
						if badCodeRegex.Match([]byte(b.Details)) {
							bouncedAt, _ := time.Parse(time.RFC3339, b.BouncedAt)
							newBlacklists = append(newBlacklists, models.EmailBlacklist{
								Email:         b.Email,
								BlacklistedAt: now,
								BouncedAt:     bouncedAt,
								Reason:        "Bounced",
								Details:       b.Details,
							})
						}
					}

					logging.Debug().Int("Num blacklisted", len(newBlacklists)).Msg("Emails to blacklist")
					tx, err := conn.Begin(ctx)
					if err != nil {
						log.Error().Err(oops.New(err, "Failed to create db transaction")).Msg("Failed to create db transaction")
						return
					}
					defer tx.Rollback(ctx)
					for _, b := range newBlacklists {
						_, err = tx.Exec(ctx,
							`
							INSERT INTO email_blacklist (email, blacklisted_at, bounced_at, reason, details)
							VALUES ($1, $2, $3, $4, $5)
							ON CONFLICT DO NOTHING
							`,
							b.Email,
							b.BlacklistedAt,
							b.BouncedAt,
							b.Reason,
							b.Details,
						)
						if err != nil {
							log.Error().Err(oops.New(err, "Failed to insert new blacklisted email")).Msg("Failed to insert new blacklisted email")
							return
						}

						_, err = tx.Exec(ctx,
							`
							UPDATE hmn_user
							SET
								status = $1,
								ban_reason = $2
							WHERE
								email = $3 AND status = ANY($4)
							`,
							models.UserStatusBanned,
							"Email blacklisted due to bounce",
							b.Email,
							[]models.UserStatus{
								models.UserStatusInactive,
								models.UserStatusConfirmed,
							},
						)
					}
					tx.Commit(ctx)
				}

				return false, nil
			}()

			if err != nil {
				log.Error().Err(err).Msg("Panicked in email bounce monitor")
			} else if done {
				return
			}
		}
	}()

	return job
}

// https://postmarkapp.com/developer/api/bounce-api#bounces
func PostmarkGetBounces(ctx context.Context, fromDate time.Time) ([]Bounce, error) {
	log := logging.ExtractLogger(ctx)

	query := url.Values{}
	query.Add("count", "500") // NOTE(asaf): Max 500 per request
	query.Add("offset", "0")
	query.Add("type", "SoftBounce")
	query.Add("inactive", "false")
	if !fromDate.IsZero() {
		location, err := time.LoadLocation("US/Eastern")
		if err == nil {
			query.Add("fromdate", fromDate.In(location).Format("2006-01-02T15:04:05"))
		} else {
			log.Warn().Err(err).Msg("Failed to load US/Eastern timezone")
		}
	}

	req, err := http.NewRequestWithContext(
		ctx,
		"GET",
		fmt.Sprintf("https://api.postmarkapp.com/bounces?%s", query.Encode()),
		nil,
	)

	if err != nil {
		return nil, oops.New(err, "failed to create postmark api request")
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Postmark-Server-Token", config.Config.Postmark.TransactionalStreamToken)

	res, err := httpClient.Do(req)
	if err != nil {
		return nil, oops.New(err, "failed to send postmark api request")
	}

	// https://postmarkapp.com/developer/api/overview#response-codes
	if res.StatusCode != 200 {
		var body []byte
		if res.StatusCode == 422 {
			// NOTE(asaf): We get additional information on 422
			body, err = io.ReadAll(res.Body)
			res.Body.Close()
			if err != nil {
				return nil, oops.New(err, "failed to read response body while processing postmark api error response")
			}
		}
		log.Error().Str("body", string(body)).Int("status", res.StatusCode).Msg("Got a bad response from postmark")

		return nil, oops.New(nil, "got a bad response from postmark")
	} else {
		body, err := io.ReadAll(res.Body)
		res.Body.Close()
		if err != nil {
			return nil, oops.New(err, "failed to read response body while processing postmark api response")
		}

		type bounceResponse struct {
			Bounces []Bounce `json:"Bounces"`
		}

		var br bounceResponse
		err = json.Unmarshal(body, &br)
		if err != nil {
			return nil, oops.New(err, "failed to parse postmark api response while processing bounces response")
		}

		return br.Bounces, nil
	}
}

package auth

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

const SessionCookieName = "HMNSession"
const CSRFFieldName = "csrf_token"

const sessionDuration = time.Hour * 24 * 14

func MakeSessionId() string {
	idBytes := make([]byte, 40)
	_, err := io.ReadFull(rand.Reader, idBytes)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(idBytes)[:40]
}

func makeCSRFToken() string {
	idBytes := make([]byte, 30)
	_, err := io.ReadFull(rand.Reader, idBytes)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(idBytes)[:30]
}

var ErrNoSession = errors.New("no session found")

func GetSession(ctx context.Context, conn *pgxpool.Pool, id string) (*models.Session, error) {
	sess, err := db.QueryOne[models.Session](ctx, conn,
		`
		SELECT $columns
		FROM session
		WHERE
			id = $1
			AND expires_at > CURRENT_TIMESTAMP
		`,
		id,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			return nil, ErrNoSession
		} else {
			return nil, oops.New(err, "failed to get session")
		}
	}

	return sess, nil
}

func CreateSession(ctx context.Context, conn *pgxpool.Pool, username string) (*models.Session, error) {
	session := models.Session{
		ID:        MakeSessionId(),
		Username:  username,
		ExpiresAt: time.Now().Add(sessionDuration),
		CSRFToken: makeCSRFToken(),
	}

	_, err := conn.Exec(ctx,
		`
		---- Create session
		INSERT INTO session (id, username, expires_at, csrf_token) VALUES ($1, $2, $3, $4)
		`,
		session.ID, session.Username, session.ExpiresAt, session.CSRFToken,
	)
	if err != nil {
		return nil, oops.New(err, "failed to persist session")
	}

	return &session, nil
}

// Deletes a session by id. If no session with that id exists, no
// error is returned.
func DeleteSession(ctx context.Context, conn *pgxpool.Pool, id string) error {
	_, err := conn.Exec(ctx, "DELETE FROM session WHERE id = $1", id)
	if err != nil {
		return oops.New(err, "failed to delete session")
	}

	return nil
}

func DeleteSessionForUser(ctx context.Context, conn *pgxpool.Pool, username string) error {
	_, err := conn.Exec(ctx,
		`
		DELETE FROM session
		WHERE LOWER(username) = LOWER($1)
		`,
		username,
	)
	if err != nil {
		return oops.New(err, "failed to delete session")
	}

	return nil
}

func NewSessionCookie(session *models.Session) *http.Cookie {
	return &http.Cookie{
		Name:  SessionCookieName,
		Value: session.ID,

		Domain:  config.Config.Auth.CookieDomain,
		Path:    "/",
		Expires: time.Now().Add(sessionDuration),

		Secure:   config.Config.Auth.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}
}

var DeleteSessionCookie = &http.Cookie{
	Name:   SessionCookieName,
	Domain: config.Config.Auth.CookieDomain,
	MaxAge: -1,
}

func DeleteExpiredSessions(ctx context.Context, conn *pgxpool.Pool) (int64, error) {
	tag, err := conn.Exec(ctx, "DELETE FROM session WHERE expires_at <= CURRENT_TIMESTAMP")
	if err != nil {
		return 0, oops.New(err, "failed to delete expired sessions")
	}

	return tag.RowsAffected(), nil
}

func DeleteExpiredPendingLogins(ctx context.Context, conn *pgxpool.Pool) (int64, error) {
	tag, err := conn.Exec(ctx, "DELETE FROM pending_login WHERE expires_at <= CURRENT_TIMESTAMP")
	if err != nil {
		return 0, oops.New(err, "failed to delete expired pending logins")
	}

	return tag.RowsAffected(), nil
}

func PeriodicallyDeleteExpiredStuff(conn *pgxpool.Pool) *jobs.Job {
	job := jobs.New("periodically delete expired stuff")
	go func() {
		defer job.Finish()

		t := time.NewTicker(1 * time.Minute)
		for {
			select {
			case <-t.C:
				err := func() (err error) {
					defer utils.RecoverPanicAsError(&err)

					n, err := DeleteExpiredSessions(job.Ctx, conn)
					if err == nil {
						if n > 0 {
							job.Logger.Info().Int64("num deleted sessions", n).Msg("Deleted expired sessions")
						}
					} else {
						job.Logger.Error().Err(err).Msg("Failed to delete expired sessions")
					}

					n, err = DeleteExpiredPendingLogins(job.Ctx, conn)
					if err == nil {
						if n > 0 {
							job.Logger.Info().Int64("num deleted pending logins", n).Msg("Deleted expired pending logins")
						}
					} else {
						job.Logger.Error().Err(err).Msg("Failed to delete expired pending logins")
					}

					return nil
				}()
				if err != nil {
					job.Logger.Error().Err(err).Msg("Panicked in PeriodicallyDeleteExpiredStuff")
				}
			case <-job.Canceled():
				return
			}
		}
	}()
	return job
}

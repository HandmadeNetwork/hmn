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
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"github.com/jackc/pgx/v4/pgxpool"
)

const SessionCookieName = "HMNSession"

const sessionDuration = time.Hour * 24 * 14

func makeSessionId() string {
	idBytes := make([]byte, 40)
	_, err := io.ReadFull(rand.Reader, idBytes)
	if err != nil {
		panic(err)
	}

	return base64.StdEncoding.EncodeToString(idBytes)[:40]
}

var ErrNoSession = errors.New("no session found")

func GetSession(ctx context.Context, conn *pgxpool.Pool, id string) (*models.Session, error) {
	var sess models.Session
	err := db.QueryOneToStruct(ctx, conn, &sess, "SELECT $columns FROM sessions WHERE id = $1", id)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return nil, ErrNoSession
		} else {
			return nil, oops.New(err, "failed to get session")
		}
	}

	return &sess, nil
}

func CreateSession(ctx context.Context, conn *pgxpool.Pool, username string) (*models.Session, error) {
	session := models.Session{
		ID:        makeSessionId(),
		Username:  username,
		ExpiresAt: time.Now().Add(sessionDuration),
	}

	_, err := conn.Exec(ctx,
		"INSERT INTO sessions (id, username, expires_at) VALUES ($1, $2, $3)",
		session.ID, session.Username, session.ExpiresAt,
	)
	if err != nil {
		return nil, oops.New(err, "failed to persist session")
	}

	return &session, nil
}

// Deletes a session by id. If no session with that id exists, no
// error is returned.
func DeleteSession(ctx context.Context, conn *pgxpool.Pool, id string) error {
	_, err := conn.Exec(ctx, "DELETE FROM sessions WHERE id = $1", id)
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
		Expires: time.Now().Add(sessionDuration),

		Secure:   config.Config.Auth.CookieSecure,
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
	}
}

var DeleteSessionCookie = &http.Cookie{
	Name:   SessionCookieName,
	Domain: config.Config.Auth.CookieDomain,
	MaxAge: -1,
}

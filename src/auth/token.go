package auth

import (
	"encoding/json"
	"net/http"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/oops"
)

const AuthCookieName = "HMNToken"

type Token struct {
	Username string `json:"username"`
}

// TODO: ENCRYPT THIS

func EncodeToken(token Token) string {
	tokenBytes, _ := json.Marshal(token)
	return string(tokenBytes)
}

func DecodeToken(tokenStr string) (Token, error) {
	var token Token
	err := json.Unmarshal([]byte(tokenStr), &token)
	if err != nil {
		// TODO: Is this worthy of an oops error, or should this just be a value handled silently by code?
		return Token{}, oops.New(err, "failed to unmarshal token")
	}

	return token, nil
}

func NewAuthCookie(username string) *http.Cookie {
	return &http.Cookie{
		Name: AuthCookieName,
		Value: EncodeToken(Token{
			Username: username,
		}),

		Domain: config.Config.CookieDomain,
		// TODO: Path?

		// Secure:   true,
		HttpOnly: true,
		SameSite: http.SameSiteDefaultMode,
	}
}

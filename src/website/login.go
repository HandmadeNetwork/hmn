package website

import (
	"errors"
	"net/http"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

func Login(c *RequestContext) ResponseData {
	// TODO: Update this endpoint to give uniform responses on errors and to be resilient to timing attacks.

	form, err := c.GetFormValues()
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, NewSafeError(err, "request must contain form data"))
	}

	username := form.Get("username")
	password := form.Get("password")
	if username == "" || password == "" {
		return ErrorResponse(http.StatusBadRequest, NewSafeError(err, "you must provide both a username and password"))
	}

	redirect := form.Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	userRow, err := db.QueryOne(c.Context(), c.Conn, models.User{}, "SELECT $columns FROM auth_user WHERE username = $1", username)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return ResponseData{
				StatusCode: http.StatusUnauthorized,
			}
		} else {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up user by username"))
		}
	}
	user := userRow.(*models.User)

	hashed, err := auth.ParsePasswordString(user.Password)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse password string"))
	}

	passwordsMatch, err := auth.CheckPassword(password, hashed)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to check password against hash"))
	}

	if passwordsMatch {
		// re-hash and save the user's password if necessary
		if hashed.IsOutdated() {
			newHashed, err := auth.HashPassword(password)
			if err == nil {
				err := auth.UpdatePassword(c.Context(), c.Conn, username, newHashed)
				if err != nil {
					c.Logger.Error().Err(err).Msg("failed to update user's password")
				}
			} else {
				c.Logger.Error().Err(err).Msg("failed to re-hash password")
			}
			// If errors happen here, we can still continue with logging them in
		}

		session, err := auth.CreateSession(c.Context(), c.Conn, username)
		if err != nil {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create session"))
		}

		res := c.Redirect(redirect, http.StatusSeeOther)
		res.SetCookie(auth.NewSessionCookie(session))

		return res
	} else {
		return c.Redirect("/", http.StatusSeeOther) // TODO: Redirect to standalone login page with error
	}
}

func Logout(c *RequestContext) ResponseData {
	sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
	if err == nil {
		// clear the session from the db immediately, no expiration
		err := auth.DeleteSession(c.Context(), c.Conn, sessionCookie.Value)
		if err != nil {
			logging.Error().Err(err).Msg("failed to delete session on logout")
		}
	}

	res := c.Redirect("/", http.StatusSeeOther) // TODO: Redirect to the page the user was currently on, or if not authorized to view that page, immediately to the home page.
	res.SetCookie(auth.DeleteSessionCookie)

	return res
}

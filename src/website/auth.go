package website

import (
	"errors"
	"fmt"
	"net/http"
	neturl "net/url"
	"regexp"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
)

var UsernameRegex = regexp.MustCompile(`^[0-9a-zA-Z][\w-]{2,29}$`)

type LoginPageData struct {
	templates.BaseData
	RedirectUrl         string
	RegisterUrl         string
	ForgotPasswordUrl   string
	LoginWithDiscordUrl string
}

func LoginPage(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return c.RejectRequest("You are already logged in.")
	}

	redirect := c.Req.URL.Query().Get("redirect")

	var res ResponseData
	res.MustWriteTemplate("auth_login.html", LoginPageData{
		BaseData:            getBaseData(c, "Log in", nil),
		RedirectUrl:         redirect,
		RegisterUrl:         hmnurl.BuildRegister(redirect),
		ForgotPasswordUrl:   hmnurl.BuildRequestPasswordReset(),
		LoginWithDiscordUrl: hmnurl.BuildLoginWithDiscord(redirect),
	}, c.Perf)
	return res
}

func Login(c *RequestContext) ResponseData {
	form, err := c.GetFormValues()
	if err != nil {
		return c.ErrorResponse(http.StatusBadRequest, NewSafeError(err, "request must contain form data"))
	}

	redirect := form.Get("redirect")

	destination := hmnurl.BuildHomepage()
	if redirect != "" && urlIsLocal(redirect) {
		destination = redirect
	}

	if c.CurrentUser != nil {
		res := c.Redirect(destination, http.StatusSeeOther)
		res.AddFutureNotice("warn", fmt.Sprintf("You are already logged in as %s.", c.CurrentUser.Username))
		return res
	}

	username := form.Get("username")
	password := form.Get("password")
	if username == "" || password == "" {
		return c.Redirect(hmnurl.BuildLoginPage(redirect), http.StatusSeeOther)
	}

	showLoginWithFailure := func(c *RequestContext, redirect string) ResponseData {
		var res ResponseData
		baseData := getBaseData(c, "Log in", nil)
		baseData.AddImmediateNotice("failure", "Incorrect username or password")
		res.MustWriteTemplate("auth_login.html", LoginPageData{
			BaseData:          baseData,
			RedirectUrl:       redirect,
			ForgotPasswordUrl: hmnurl.BuildRequestPasswordReset(),
		}, c.Perf)
		return res
	}

	user, err := db.QueryOne[models.User](c, c.Conn,
		`
		SELECT $columns
		FROM hmn_user
		WHERE LOWER(username) = LOWER($1)
		`,
		username,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			return showLoginWithFailure(c, redirect)
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up user by username"))
		}
	}

	success, err := tryLogin(c, user, password)

	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if !success {
		return showLoginWithFailure(c, redirect)
	}

	if user.Status == models.UserStatusInactive {
		return c.RejectRequest("You must validate your email address before logging in. You should've received an email shortly after registration. If you did not receive the email, please contact the staff.")
	}

	res := c.Redirect(destination, http.StatusSeeOther)
	err = loginUser(c, user, &res)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}
	return res
}

func LoginWithDiscord(c *RequestContext) ResponseData {
	destinationUrl := c.URL().Query().Get("redirect")
	if c.CurrentUser != nil {
		return c.Redirect(destinationUrl, http.StatusSeeOther)
	}

	pendingLogin, err := db.QueryOne[models.PendingLogin](c, c.Conn,
		`
		INSERT INTO pending_login (id, expires_at, destination_url)
		VALUES ($1, $2, $3)
		RETURNING $columns
		`,
		auth.MakeSessionId(), time.Now().Add(time.Minute*10), destinationUrl,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save pending login"))
	}

	discordAuthUrl := discord.GetAuthorizeUrl(pendingLogin.ID, true)
	return c.Redirect(discordAuthUrl, http.StatusSeeOther)
}

func Logout(c *RequestContext) ResponseData {
	redirect := c.Req.URL.Query().Get("redirect")

	destination := hmnurl.BuildHomepage()
	if redirect != "" && urlIsLocal(redirect) {
		destination = redirect
	}

	res := c.Redirect(destination, http.StatusSeeOther)
	logoutUser(c, &res)
	return res
}

func RegisterNewUser(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		c.Redirect(hmnurl.BuildUserSettings(c.CurrentUser.Username), http.StatusSeeOther)
	}

	// TODO(asaf): Do something to prevent bot registration

	type RegisterPageData struct {
		templates.BaseData
		DestinationURL string
	}

	tmpl := RegisterPageData{
		BaseData:       getBaseData(c, "Register", nil),
		DestinationURL: c.Req.URL.Query().Get("destination"),
	}

	var res ResponseData
	res.MustWriteTemplate("auth_register.html", tmpl, c.Perf)
	return res
}

func RegisterNewUserSubmit(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return c.RejectRequest("Can't register new user. You are already logged in")
	}
	c.Req.ParseForm()

	username := strings.TrimSpace(c.Req.Form.Get("username"))
	displayName := strings.TrimSpace(c.Req.Form.Get("displayname"))
	emailAddress := strings.TrimSpace(c.Req.Form.Get("email"))
	password := c.Req.Form.Get("password")
	destination := strings.TrimSpace(c.Req.Form.Get("destination"))

	logEvent := c.Logger.Info().
		Str("username", username).
		Str("email", emailAddress).
		Str("ip", c.Req.RemoteAddr).
		Str("Referer", c.Req.Referer()).
		Str("X-Forwarded-For", c.Req.Header.Get("X-Forwarded-For"))

	if !UsernameRegex.Match([]byte(username)) {
		logEvent.Msg("registration attempt with invalid username")
		return c.RejectRequest("Invalid username")
	}
	if !email.IsEmail(emailAddress) {
		logEvent.Msg("registration attempt with invalid email address")
		return c.RejectRequest("Invalid email address")
	}
	if len(password) < 8 {
		logEvent.Msg("registration attempt with invalid password")
		return c.RejectRequest("Password too short")
	}

	c.Perf.StartBlock("SQL", "Check blacklist")
	if blacklist(username, emailAddress) {
		// NOTE(asaf): Silent rejection so we don't allow attackers to harvest emails.
		logEvent.Msg("blacklisted registration attempt")
		return c.Redirect(hmnurl.BuildRegistrationSuccess(), http.StatusSeeOther)
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Check for existing usernames and emails")
	userAlreadyExists := true
	_, err := db.QueryOneScalar[int](c, c.Conn,
		`
		SELECT id
		FROM hmn_user
		WHERE LOWER(username) = LOWER($1)
		`,
		username,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			userAlreadyExists = false
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user"))
		}
	}

	if userAlreadyExists {
		logEvent.Msg("registration attempt with duplicate username")
		return c.RejectRequest(fmt.Sprintf("Username (%s) already exists.", username))
	}

	existingUser, err := db.QueryOne[models.User](c, c.Conn,
		`
		SELECT $columns
		FROM hmn_user
		WHERE LOWER(email) = LOWER($1)
		`,
		emailAddress,
	)
	if errors.Is(err, db.NotFound) {
		// this is fine
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user"))
	}
	c.Perf.EndBlock()

	if existingUser != nil {
		// Render the page as if it was a successful new registration, but
		// instead send an email to the duplicate email address containing
		// their actual username. Spammers won't be able to harvest emails, but
		// normal users will be able to find and access their old accounts.

		err := email.SendExistingAccountEmail(
			existingUser.Email,
			existingUser.BestName(),
			existingUser.Username,
			destination,
			c.Perf,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to send existing account email"))
		}

		logEvent.Msg("registration attempt with duplicate email (follow-up sent)")
		return c.Redirect(hmnurl.BuildRegistrationSuccess(), http.StatusSeeOther)
	}

	hashed := auth.HashPassword(password)

	c.Perf.StartBlock("SQL", "Create user and one time token")
	tx, err := c.Conn.Begin(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to start db transaction"))
	}
	defer tx.Rollback(c)

	now := time.Now()

	var newUserId int
	err = tx.QueryRow(c,
		`
		INSERT INTO hmn_user (username, email, password, date_joined, name, registration_ip)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
		`,
		username, emailAddress, hashed.String(), now, displayName, c.GetIP(),
	).Scan(&newUserId)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to store user"))
	}

	ott := models.GenerateToken()
	_, err = tx.Exec(c,
		`
		INSERT INTO one_time_token (token_type, created, expires, token_content, owner_id)
		VALUES($1, $2, $3, $4, $5)
		`,
		models.TokenTypeRegistration,
		now,
		now.Add(time.Hour*24*7),
		ott,
		newUserId,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to store one-time token"))
	}
	c.Perf.EndBlock()

	mailName := displayName
	if mailName == "" {
		mailName = username
	}
	err = email.SendRegistrationEmail(
		emailAddress,
		mailName,
		username,
		ott,
		destination,
		c.Perf,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to send registration email"))
	}

	if config.Config.Env == config.Dev {
		confirmUrl := hmnurl.BuildEmailConfirmation(username, ott, destination)
		logging.Debug().Str("Confirmation url", confirmUrl).Msg("New user requires email confirmation")
	}

	c.Perf.StartBlock("SQL", "Commit user")
	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit user to the db"))
	}
	c.Perf.EndBlock()

	logEvent.Msg("registration succeeded")
	return c.Redirect(hmnurl.BuildRegistrationSuccess(), http.StatusSeeOther)
}

type RegisterNewUserSuccessData struct {
	templates.BaseData
	ContactUsUrl string
}

func RegisterNewUserSuccess(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
	}

	var res ResponseData
	res.MustWriteTemplate("auth_register_success.html", RegisterNewUserSuccessData{
		BaseData:     getBaseData(c, "Register", nil),
		ContactUsUrl: hmnurl.BuildContactPage(),
	}, c.Perf)
	return res
}

type EmailValidationData struct {
	templates.BaseData
	Token          string
	Username       string
	DestinationURL string
}

func EmailConfirmation(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
	}

	username, hasUsername := c.PathParams["username"]
	if !hasUsername {
		return c.RejectRequest("Bad validation url")
	}

	token := ""
	hasToken := false

	// TODO(asaf): Delete old hash/nonce about a week after launch
	hash, hasHash := c.PathParams["hash"]
	nonce, hasNonce := c.PathParams["nonce"]
	if hasHash && hasNonce {
		token = fmt.Sprintf("%s/%s", hash, nonce)
		hasToken = true
	} else {
		token, hasToken = c.PathParams["token"]
	}

	if !hasToken {
		return c.RejectRequest("Bad validation url")
	}

	validationResult := validateUsernameAndToken(c, username, token, models.TokenTypeRegistration)
	if !validationResult.Match {
		return makeResponseForBadRegistrationTokenValidationResult(c, validationResult)
	}

	var res ResponseData
	res.MustWriteTemplate("auth_email_validation.html", EmailValidationData{
		BaseData:       getBaseData(c, "Register", nil),
		Token:          token,
		Username:       username,
		DestinationURL: c.Req.URL.Query().Get("destination"),
	}, c.Perf)
	return res
}

func EmailConfirmationSubmit(c *RequestContext) ResponseData {
	c.Req.ParseForm()

	token := c.Req.Form.Get("token")
	username := c.Req.Form.Get("username")
	password := c.Req.Form.Get("password")
	destination := c.Req.Form.Get("destination")

	validationResult := validateUsernameAndToken(c, username, token, models.TokenTypeRegistration)
	if !validationResult.Match {
		return makeResponseForBadRegistrationTokenValidationResult(c, validationResult)
	}

	success, err := tryLogin(c, validationResult.User, password)

	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	} else if !success {
		var res ResponseData
		baseData := getBaseData(c, "Register", nil)
		// NOTE(asaf): We can report that the password is incorrect, because an attacker wouldn't have a valid token to begin with.
		baseData.AddImmediateNotice("failure", "Incorrect password. Please try again.")
		res.MustWriteTemplate("auth_email_validation.html", EmailValidationData{
			BaseData:       baseData,
			Token:          token,
			Username:       username,
			DestinationURL: destination,
		}, c.Perf)
		return res
	}

	c.Perf.StartBlock("SQL", "Updating user status and deleting token")
	tx, err := c.Conn.Begin(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to start db transaction"))
	}
	defer tx.Rollback(c)

	_, err = tx.Exec(c,
		`
		UPDATE hmn_user
		SET status = $1
		WHERE id = $2
		`,
		models.UserStatusConfirmed,
		validationResult.User.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update user status"))
	}

	_, err = tx.Exec(c,
		`
		DELETE FROM one_time_token WHERE id = $1
		`,
		validationResult.OneTimeToken.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete one time token"))
	}

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit transaction"))
	}
	c.Perf.EndBlock()

	redirect := hmnurl.BuildHomepage()
	if destination != "" && urlIsLocal(destination) {
		redirect = destination
	}

	res := c.Redirect(redirect, http.StatusSeeOther)
	res.AddFutureNotice("success", "You've completed your registration successfully!")
	err = loginUser(c, validationResult.User, &res)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}
	return res
}

// NOTE(asaf): Only call this when validationResult.Match is false.
func makeResponseForBadRegistrationTokenValidationResult(c *RequestContext, validationResult validateUserAndTokenResult) ResponseData {
	if validationResult.User == nil {
		return c.RejectRequest("You haven't validated your email in time and your user was deleted. You may try registering again with the same username.")
	}

	if validationResult.OneTimeToken == nil {
		// NOTE(asaf): The user exists, but the validation token doesn't.
		//			   That means the user already validated their email and can just log in normally.
		return c.Redirect(hmnurl.BuildLoginPage(""), http.StatusSeeOther)
	}

	return c.RejectRequest("Bad token. If you are having problems registering or logging in, please contact the staff.")
}

// NOTE(asaf): PasswordReset refers specifically to "forgot your password" flow over email,
//
//	not to changing your password through the user settings page.
func RequestPasswordReset(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
	}
	var res ResponseData
	res.MustWriteTemplate("auth_password_reset.html", getBaseData(c, "Password Reset", nil), c.Perf)
	return res
}

func RequestPasswordResetSubmit(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
	}
	c.Req.ParseForm()

	username := strings.TrimSpace(c.Req.Form.Get("username"))
	emailAddress := strings.TrimSpace(c.Req.Form.Get("email"))

	if username == "" && emailAddress == "" {
		return c.RejectRequest("You must provide a username and an email address.")
	}

	c.Perf.StartBlock("SQL", "Fetching user")
	type userQuery struct {
		User models.User `db:"hmn_user"`
	}
	user, err := db.QueryOne[models.User](c, c.Conn,
		`
		SELECT $columns
		FROM hmn_user
		WHERE
			LOWER(username) = LOWER($1)
			AND LOWER(email) = LOWER($2)
		`,
		username,
		emailAddress,
	)
	c.Perf.EndBlock()
	if err != nil {
		if !errors.Is(err, db.NotFound) {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up user by username"))
		}
	}

	if user != nil && user.Status != models.UserStatusBanned {
		c.Perf.StartBlock("SQL", "Fetching existing token")
		resetToken, err := db.QueryOne[models.OneTimeToken](c, c.Conn,
			`
			SELECT $columns
			FROM one_time_token
			WHERE
				token_type = $1
				AND owner_id = $2
			`,
			models.TokenTypePasswordReset,
			user.ID,
		)
		c.Perf.EndBlock()
		if err != nil {
			if !errors.Is(err, db.NotFound) {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch onetimetoken for user"))
			}
		}
		now := time.Now()

		if resetToken != nil {
			if resetToken.Expires.Before(now.Add(time.Minute * 30)) { // NOTE(asaf): Expired or about to expire
				c.Perf.StartBlock("SQL", "Deleting expired token")
				_, err = c.Conn.Exec(c,
					`
					DELETE FROM one_time_token
					WHERE id = $1
					`,
					resetToken.ID,
				)
				c.Perf.EndBlock()
				if err != nil {
					return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete onetimetoken"))
				}
				resetToken = nil
			}
		}

		if resetToken == nil {
			c.Perf.StartBlock("SQL", "Creating new token")
			newToken, err := db.QueryOne[models.OneTimeToken](c, c.Conn,
				`
				INSERT INTO one_time_token (token_type, created, expires, token_content, owner_id)
				VALUES ($1, $2, $3, $4, $5)
				RETURNING $columns
				`,
				models.TokenTypePasswordReset,
				now,
				now.Add(time.Hour*24),
				models.GenerateToken(),
				user.ID,
			)
			c.Perf.EndBlock()
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create onetimetoken"))
			}
			resetToken = newToken

			err = email.SendPasswordReset(
				user.Email,
				user.BestName(),
				user.Username,
				resetToken.Content,
				resetToken.Expires,
				c.Perf,
			)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to send email"))
			}

			if config.Config.Env == config.Dev {
				passwordResetUrl := hmnurl.BuildDoPasswordReset(username, resetToken.Content)
				logging.Debug().Str("Reset url", passwordResetUrl).Msg("Password reset requested")
			}
		}
	}
	return c.Redirect(hmnurl.BuildPasswordResetSent(), http.StatusSeeOther)
}

type PasswordResetSentData struct {
	templates.BaseData
	ContactUsUrl string
}

func PasswordResetSent(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
	}
	var res ResponseData
	res.MustWriteTemplate("auth_password_reset_sent.html", PasswordResetSentData{
		BaseData:     getBaseData(c, "Password Reset", nil),
		ContactUsUrl: hmnurl.BuildContactPage(),
	}, c.Perf)
	return res
}

type DoPasswordResetData struct {
	templates.BaseData
	Username string
	Token    string
}

func DoPasswordReset(c *RequestContext) ResponseData {
	username, hasUsername := c.PathParams["username"]
	token, hasToken := c.PathParams["token"]

	if !hasToken || !hasUsername {
		return c.RejectRequest("Bad validation url.")
	}

	validationResult := validateUsernameAndToken(c, username, token, models.TokenTypePasswordReset)
	if !validationResult.Match {
		return c.RejectRequest("Bad validation url.")
	}

	var res ResponseData

	if c.CurrentUser != nil && c.CurrentUser.ID != validationResult.User.ID {
		// NOTE(asaf): In the rare case that a user is logged in with user A and is trying to
		//             change the password for user B, log out the current user to avoid confusion.
		logoutUser(c, &res)
	}

	res.MustWriteTemplate("auth_do_password_reset.html", DoPasswordResetData{
		BaseData: getBaseData(c, "Password Reset", nil),
		Username: username,
		Token:    token,
	}, c.Perf)
	return res
}

func DoPasswordResetSubmit(c *RequestContext) ResponseData {
	c.Req.ParseForm()

	token := c.Req.Form.Get("token")
	username := c.Req.Form.Get("username")
	password := c.Req.Form.Get("password")

	validationResult := validateUsernameAndToken(c, username, token, models.TokenTypePasswordReset)
	if !validationResult.Match {
		return c.RejectRequest("Bad validation url.")
	}

	if c.CurrentUser != nil && c.CurrentUser.ID != validationResult.User.ID {
		return c.RejectRequest(fmt.Sprintf("Can't change password for %s. You are logged in as %s.", username, c.CurrentUser.Username))
	}

	if len(password) < 8 {
		return c.RejectRequest("Password too short")
	}

	hashed := auth.HashPassword(password)

	c.Perf.StartBlock("SQL", "Update user's password and delete reset token")
	tx, err := c.Conn.Begin(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to start db transaction"))
	}
	defer tx.Rollback(c)

	tag, err := tx.Exec(c,
		`
		UPDATE hmn_user
		SET password = $1
		WHERE id = $2
		`,
		hashed.String(),
		validationResult.User.ID,
	)
	if err != nil || tag.RowsAffected() == 0 {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update user's password"))
	}

	if validationResult.User.Status == models.UserStatusInactive {
		_, err = tx.Exec(c,
			`
			UPDATE hmn_user
			SET status = $1
			WHERE id = $2
			`,
			models.UserStatusConfirmed,
			validationResult.User.ID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update user's status"))
		}
	}

	_, err = tx.Exec(c,
		`
		DELETE FROM one_time_token
		WHERE id = $1
		`,
		validationResult.OneTimeToken.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete onetimetoken"))
	}

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit password reset to the db"))
	}
	c.Perf.EndBlock()

	res := c.Redirect(hmnurl.BuildUserSettings(""), http.StatusSeeOther)
	res.AddFutureNotice("success", "Password changed successfully.")
	err = loginUser(c, validationResult.User, &res)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}
	return res
}

func tryLogin(c *RequestContext, user *models.User, password string) (bool, error) {
	if user.Status == models.UserStatusBanned {
		return false, nil
	}

	c.Perf.StartBlock("AUTH", "Checking password")
	defer c.Perf.EndBlock()
	hashed, err := auth.ParsePasswordString(user.Password)
	if err != nil {
		if user.Password == "" {
			return false, nil
		} else {
			return false, oops.New(err, "failed to parse password string")
		}
	}

	passwordsMatch, err := auth.CheckPassword(password, hashed)
	if err != nil {
		return false, oops.New(err, "failed to check password against hash")
	}

	if !passwordsMatch {
		return false, nil
	}

	// re-hash and save the user's password if necessary
	if hashed.IsOutdated() {
		newHashed := auth.HashPassword(password)
		err := auth.UpdatePassword(c, c.Conn, user.Username, newHashed)
		if err != nil {
			c.Logger.Error().Err(err).Msg("failed to update user's password")
		}
		// If errors happen here, we can still continue with logging them in
	}

	return true, nil
}

func loginUser(c *RequestContext, user *models.User, responseData *ResponseData) error {
	c.Perf.StartBlock("SQL", "Setting last login and creating session")
	defer c.Perf.EndBlock()
	tx, err := c.Conn.Begin(c)
	if err != nil {
		return oops.New(err, "failed to start db transaction")
	}
	defer tx.Rollback(c)

	now := time.Now()

	_, err = tx.Exec(c,
		`
		UPDATE hmn_user
		SET last_login = $1
		WHERE id = $2
		`,
		now,
		user.ID,
	)
	if err != nil {
		return oops.New(err, "failed to update last_login for user")
	}

	session, err := auth.CreateSession(c, c.Conn, user.Username)
	if err != nil {
		return oops.New(err, "failed to create session")
	}

	err = tx.Commit(c)
	if err != nil {
		return oops.New(err, "failed to commit transaction")
	}
	responseData.SetCookie(auth.NewSessionCookie(session))
	return nil
}

func logoutUser(c *RequestContext, res *ResponseData) {
	sessionCookie, err := c.Req.Cookie(auth.SessionCookieName)
	if err == nil {
		// clear the session from the db immediately, no expiration
		err := auth.DeleteSession(c, c.Conn, sessionCookie.Value)
		if err != nil {
			logging.Error().Err(err).Msg("failed to delete session on logout")
		}
	}

	res.SetCookie(auth.DeleteSessionCookie)
}

type validateUserAndTokenResult struct {
	User         *models.User
	OneTimeToken *models.OneTimeToken
	Match        bool
	Error        error
}

func validateUsernameAndToken(c *RequestContext, username string, token string, tokenType models.OneTimeTokenType) validateUserAndTokenResult {
	c.Perf.StartBlock("SQL", "Check username and token")
	defer c.Perf.EndBlock()
	type userAndTokenQuery struct {
		User         models.User          `db:"hmn_user"`
		OneTimeToken *models.OneTimeToken `db:"onetimetoken"`
	}
	data, err := db.QueryOne[userAndTokenQuery](c, c.Conn,
		`
		SELECT $columns
		FROM hmn_user
		LEFT JOIN asset AS hmn_user_avatar ON hmn_user_avatar.id = hmn_user.avatar_asset_id
		LEFT JOIN one_time_token AS onetimetoken ON onetimetoken.owner_id = hmn_user.id
		WHERE
			LOWER(hmn_user.username) = LOWER($1)
			AND onetimetoken.token_type = $2
		`,
		username,
		tokenType,
	)
	var result validateUserAndTokenResult
	if err != nil {
		if !errors.Is(err, db.NotFound) {
			result.Error = oops.New(err, "failed to fetch user and token from db")
			return result
		}
	}
	if data != nil {
		result.User = &data.User
		result.OneTimeToken = data.OneTimeToken
		if result.OneTimeToken != nil {
			result.Match = (result.OneTimeToken.Content == token)
		}
	}

	return result
}

func urlIsLocal(url string) bool {
	urlParsed, err := neturl.Parse(url)
	if err != nil {
		return false
	}
	baseUrl := utils.Must1(neturl.Parse(config.Config.BaseUrl))
	return strings.HasSuffix(urlParsed.Host, baseUrl.Host)
}

var reStupidUsername = regexp.MustCompile(`[a-z]{10}`)

func blacklist(username, email string) bool {
	if reStupidUsername.MatchString(username) {
		return true
	}
	if strings.Count(email, ".") > 5 {
		return true
	}

	// TODO(asaf): Actually check email against blacklist

	return false
}

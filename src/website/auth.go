package website

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"regexp"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

// TODO(asaf): Add a middleware that guarantees the certain handlers will take at least X amount of time.
//             Will be relevant for:
//             * Login POST
//             * Register POST

var UsernameRegex = regexp.MustCompile(`^[0-9a-zA-Z][\w-]{2,29}$`)

type LoginPageData struct {
	templates.BaseData
	RedirectUrl       string
	ForgotPasswordUrl string
}

func LoginPage(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return RejectRequest(c, "You are already logged in.")
	}

	var res ResponseData
	res.MustWriteTemplate("auth_login.html", LoginPageData{
		BaseData:          getBaseData(c),
		RedirectUrl:       c.Req.URL.Query().Get("redirect"),
		ForgotPasswordUrl: hmnurl.BuildPasswordResetRequest(),
	}, c.Perf)
	return res
}

func Login(c *RequestContext) ResponseData {
	// TODO: Update this endpoint to give uniform responses on errors and to be resilient to timing attacks.
	if c.CurrentUser != nil {
		return RejectRequest(c, "You are already logged in.")
	}

	form, err := c.GetFormValues()
	if err != nil {
		return ErrorResponse(http.StatusBadRequest, NewSafeError(err, "request must contain form data"))
	}

	username := form.Get("username")
	password := form.Get("password")
	if username == "" || password == "" {
		return RejectRequest(c, "You must provide both a username and password")
	}

	redirect := form.Get("redirect")
	if redirect == "" {
		redirect = "/"
	}

	userRow, err := db.QueryOne(c.Context(), c.Conn, models.User{}, "SELECT $columns FROM auth_user WHERE LOWER(username) = LOWER($1)", username)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			var res ResponseData
			baseData := getBaseData(c)
			baseData.Notices = []templates.Notice{{Content: "Incorrect username or password", Class: "failure"}}
			res.MustWriteTemplate("auth_login.html", LoginPageData{
				BaseData:          baseData,
				RedirectUrl:       redirect,
				ForgotPasswordUrl: hmnurl.BuildPasswordResetRequest(),
			}, c.Perf)
			return res
		} else {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up user by username"))
		}
	}
	user := userRow.(*models.User)

	success, err := tryLogin(c, user, password)

	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, err)
	}

	if !success {
		var res ResponseData
		baseData := getBaseData(c)
		baseData.Notices = []templates.Notice{{Content: "Incorrect username or password", Class: "failure"}}
		res.MustWriteTemplate("auth_login.html", LoginPageData{
			BaseData:          baseData,
			RedirectUrl:       redirect,
			ForgotPasswordUrl: hmnurl.BuildPasswordResetRequest(),
		}, c.Perf)
		return res
	}

	if user.Status == models.UserStatusInactive {
		return RejectRequest(c, "You must validate your email address before logging in. You should've received an email shortly after registration. If you did not receive the email, please contact the staff.")
	}

	res := c.Redirect(redirect, http.StatusSeeOther)
	err = loginUser(c, user, &res)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, err)
	}
	return res
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

	redir := c.Req.URL.Query().Get("redirect")
	if redir == "" {
		redir = "/"
	}

	res := c.Redirect(redir, http.StatusSeeOther)
	res.SetCookie(auth.DeleteSessionCookie)

	return res
}

func RegisterNewUser(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		c.Redirect(hmnurl.BuildUserSettings(c.CurrentUser.Username), http.StatusSeeOther)
	}
	// TODO(asaf): Do something to prevent bot registration
	var res ResponseData
	res.MustWriteTemplate("auth_register.html", getBaseData(c), c.Perf)
	return res
}

func RegisterNewUserSubmit(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return RejectRequest(c, "Can't register new user. You are already logged in")
	}
	c.Req.ParseForm()

	username := strings.TrimSpace(c.Req.Form.Get("username"))
	displayName := strings.TrimSpace(c.Req.Form.Get("displayname"))
	emailAddress := strings.TrimSpace(c.Req.Form.Get("email"))
	password := c.Req.Form.Get("password")
	password2 := c.Req.Form.Get("password2")
	if !UsernameRegex.Match([]byte(username)) {
		return RejectRequest(c, "Invalid username")
	}
	if !email.IsEmail(emailAddress) {
		return RejectRequest(c, "Invalid email address")
	}
	if len(password) < 8 {
		return RejectRequest(c, "Password too short")
	}
	if password != password2 {
		return RejectRequest(c, "Password confirmation doesn't match password")
	}

	c.Perf.StartBlock("SQL", "Check for existing usernames and emails")
	userAlreadyExists := true
	_, err := db.QueryInt(c.Context(), c.Conn,
		`
		SELECT id
		FROM auth_user
		WHERE LOWER(username) = LOWER($1)
		`,
		username,
	)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			userAlreadyExists = false
		} else {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user"))
		}
	}

	if userAlreadyExists {
		return RejectRequest(c, fmt.Sprintf("Username (%s) already exists.", username))
	}

	emailAlreadyExists := true
	_, err = db.QueryInt(c.Context(), c.Conn,
		`
		SELECT id
		FROM auth_user
		WHERE LOWER(email) = LOWER($1)
		`,
		emailAddress,
	)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			emailAlreadyExists = false
		} else {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user"))
		}
	}
	c.Perf.EndBlock()

	if emailAlreadyExists {
		// NOTE(asaf): Silent rejection so we don't allow attackers to harvest emails.
		time.Sleep(time.Second * 3) // NOTE(asaf): Pretend to send email
		return c.Redirect(hmnurl.BuildRegistrationSuccess(), http.StatusSeeOther)
	}

	hashed, err := auth.HashPassword(password)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to encrypt password"))
	}

	c.Perf.StartBlock("SQL", "Check blacklist")
	// TODO(asaf): Check email against blacklist
	blacklisted := false
	if blacklisted {
		// NOTE(asaf): Silent rejection so we don't allow attackers to harvest emails.
		time.Sleep(time.Second * 3) // NOTE(asaf): Pretend to send email
		return c.Redirect(hmnurl.BuildRegistrationSuccess(), http.StatusSeeOther)
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Create user and one time token")
	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to start db transaction"))
	}
	defer tx.Rollback(c.Context())

	now := time.Now()

	var newUserId int
	err = tx.QueryRow(c.Context(),
		`
		INSERT INTO auth_user (username, email, password, date_joined, name, registration_ip)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
		`,
		username, emailAddress, hashed.String(), now, displayName, c.GetIP(),
	).Scan(&newUserId)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to store user"))
	}

	ott := models.GenerateToken()
	_, err = tx.Exec(c.Context(),
		`
		INSERT INTO handmade_onetimetoken (token_type, created, expires, token_content, owner_id)
		VALUES($1, $2, $3, $4, $5)
		`,
		models.TokenTypeRegistration,
		now,
		now.Add(time.Hour*24*7),
		ott,
		newUserId,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to store one-time token"))
	}
	c.Perf.EndBlock()

	mailName := displayName
	if mailName == "" {
		mailName = username
	}
	err = email.SendRegistrationEmail(emailAddress, mailName, username, ott, c.Perf)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to send registration email"))
	}

	c.Perf.StartBlock("SQL", "Commit user")
	err = tx.Commit(c.Context())
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit user to the db"))
	}
	c.Perf.EndBlock()
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
		BaseData:     getBaseData(c),
		ContactUsUrl: hmnurl.BuildContactPage(),
	}, c.Perf)
	return res
}

type EmailValidationData struct {
	templates.BaseData
	Token    string
	Username string
}

func EmailConfirmation(c *RequestContext) ResponseData {
	if c.CurrentUser != nil {
		return c.Redirect(hmnurl.BuildHomepage(), http.StatusSeeOther)
	}

	username, hasUsername := c.PathParams["username"]
	if !hasUsername {
		return RejectRequest(c, "Bad validation url")
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
		return RejectRequest(c, "Bad validation url")
	}

	validationResult := validateUsernameAndToken(c, username, token)
	if !validationResult.IsValid {
		return validationResult.InvalidResponse
	}

	var res ResponseData
	res.MustWriteTemplate("auth_email_validation.html", EmailValidationData{
		BaseData: getBaseData(c),
		Token:    token,
		Username: username,
	}, c.Perf)
	return res
}

func EmailConfirmationSubmit(c *RequestContext) ResponseData {
	c.Req.ParseForm()

	token := c.Req.Form.Get("token")
	username := c.Req.Form.Get("username")
	password := c.Req.Form.Get("password")

	validationResult := validateUsernameAndToken(c, username, token)
	if !validationResult.IsValid {
		return validationResult.InvalidResponse
	}

	success, err := tryLogin(c, validationResult.User, password)

	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, err)
	} else if !success {
		var res ResponseData
		baseData := getBaseData(c)
		// NOTE(asaf): We can report that the password is incorrect, because an attacker wouldn't have a valid token to begin with.
		baseData.Notices = []templates.Notice{{Content: template.HTML("Incorrect password. Please try again."), Class: "failure"}}
		res.MustWriteTemplate("auth_email_validation.html", EmailValidationData{
			BaseData: getBaseData(c),
			Token:    token,
			Username: username,
		}, c.Perf)
		return res
	}

	c.Perf.StartBlock("SQL", "Updating user status and deleting token")
	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to start db transaction"))
	}
	defer tx.Rollback(c.Context())

	_, err = tx.Exec(c.Context(),
		`
		UPDATE auth_user
		SET status = $1
		WHERE id = $2
		`,
		models.UserStatusActive,
		validationResult.User.ID,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update user status"))
	}

	_, err = tx.Exec(c.Context(),
		`
		DELETE FROM handmade_onetimetoken WHERE id = $1
		`,
		validationResult.OneTimeToken.ID,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete one time token"))
	}

	err = tx.Commit(c.Context())
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit transaction"))
	}
	c.Perf.EndBlock()

	res := c.Redirect(hmnurl.BuildHomepageWithRegistrationSuccess(), http.StatusSeeOther)
	err = loginUser(c, validationResult.User, &res)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, err)
	}
	return res
}

// NOTE(asaf): PasswordReset refers specifically to "forgot your password" flow over email,
//             not to changing your password through the user settings page.
func RequestPasswordReset(c *RequestContext) ResponseData {
	// Verify not logged in
	// Show form
	return FourOhFour(c)
}

func RequestPasswordResetSubmit(c *RequestContext) ResponseData {
	// Verify not logged in
	// Verify email input
	// Potentially send password reset email if no password reset request is active
	// Show success page in all cases
	return FourOhFour(c)
}

func PasswordReset(c *RequestContext) ResponseData {
	// Verify reset token
	// If logged in
	//   If same user as reset request -> Mark password reset as resolved and redirect to user settings
	//   If other user -> Log out
	// Show form
	return FourOhFour(c)
}

func PasswordResetSubmit(c *RequestContext) ResponseData {
	// Verify not logged in
	// Verify reset token
	// Verify not resolved
	// Verify inputs
	// Set new password
	// Mark resolved
	// Log in
	// Redirect to user settings page
	return FourOhFour(c)
}

func tryLogin(c *RequestContext, user *models.User, password string) (bool, error) {
	c.Perf.StartBlock("AUTH", "Checking password")
	defer c.Perf.EndBlock()
	hashed, err := auth.ParsePasswordString(user.Password)
	if err != nil {
		return false, oops.New(err, "failed to parse password string")
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
		newHashed, err := auth.HashPassword(password)
		if err == nil {
			err := auth.UpdatePassword(c.Context(), c.Conn, user.Username, newHashed)
			if err != nil {
				c.Logger.Error().Err(err).Msg("failed to update user's password")
			}
		} else {
			c.Logger.Error().Err(err).Msg("failed to re-hash password")
		}
		// If errors happen here, we can still continue with logging them in
	}

	return true, nil
}

func loginUser(c *RequestContext, user *models.User, responseData *ResponseData) error {
	c.Perf.StartBlock("SQL", "Setting last login and creating session")
	defer c.Perf.EndBlock()
	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		return oops.New(err, "failed to start db transaction")
	}
	defer tx.Rollback(c.Context())

	now := time.Now()

	_, err = tx.Exec(c.Context(),
		`
		UPDATE auth_user
		SET last_login = $1
		WHERE id = $2
		`,
		now,
		user.ID,
	)
	if err != nil {
		return oops.New(err, "failed to update last_login for user")
	}

	session, err := auth.CreateSession(c.Context(), c.Conn, user.Username)
	if err != nil {
		return oops.New(err, "failed to create session")
	}

	err = tx.Commit(c.Context())
	if err != nil {
		return oops.New(err, "failed to commit transaction")
	}
	responseData.SetCookie(auth.NewSessionCookie(session))
	return nil
}

type validateUserAndTokenResult struct {
	User            *models.User
	OneTimeToken    *models.OneTimeToken
	IsValid         bool
	InvalidResponse ResponseData
}

func validateUsernameAndToken(c *RequestContext, username string, token string) validateUserAndTokenResult {
	c.Perf.StartBlock("SQL", "Check username and token")
	defer c.Perf.EndBlock()
	type userAndTokenQuery struct {
		User         models.User          `db:"auth_user"`
		OneTimeToken *models.OneTimeToken `db:"onetimetoken"`
	}
	row, err := db.QueryOne(c.Context(), c.Conn, userAndTokenQuery{},
		`
		SELECT $columns
		FROM auth_user
		LEFT JOIN handmade_onetimetoken AS onetimetoken ON onetimetoken.owner_id = auth_user.id
		WHERE LOWER(auth_user.username) = LOWER($1)
		`,
		username,
	)
	var result validateUserAndTokenResult
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			result.IsValid = false
			result.InvalidResponse = RejectRequest(c, "You haven't validated your email in time and your user was deleted. You may try registering again with the same username.")
		} else {
			result.IsValid = false
			result.InvalidResponse = ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch token from db"))
		}
	} else {
		data := row.(*userAndTokenQuery)
		if data.OneTimeToken == nil {
			// NOTE(asaf): The user exists, but the validation token doesn't. That means the user already validated their email and can just log in normally.
			result.IsValid = false
			result.InvalidResponse = c.Redirect(hmnurl.BuildLoginPage(""), http.StatusSeeOther)
		} else if data.OneTimeToken.Content != token {
			result.IsValid = false
			result.InvalidResponse = RejectRequest(c, "Bad token. If you are having problems registering or logging in, please contact the staff.")
		} else {
			result.IsValid = true
			result.User = &data.User
			result.OneTimeToken = data.OneTimeToken
		}
	}

	return result
}

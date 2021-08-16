package website

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
)

func DiscordTest(c *RequestContext) ResponseData {
	var userDiscord *models.DiscordUser
	iUserDiscord, err := db.QueryOne(c.Context(), c.Conn, models.DiscordUser{},
		`
		SELECT $columns
		FROM handmade_discorduser
		WHERE hmn_user_id = $1
		`,
		c.CurrentUser.ID,
	)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			// we're ok, just no user
		} else {
			return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch current user's Discord account"))
		}
	} else {
		userDiscord = iUserDiscord.(*models.DiscordUser)
	}

	type templateData struct {
		templates.BaseData
		DiscordUser  *templates.DiscordUser
		AuthorizeURL string
	}

	baseData := getBaseData(c)
	baseData.Title = "Discord Test"

	params := make(url.Values)
	params.Set("response_type", "code")
	params.Set("client_id", config.Config.Discord.OAuthClientID)
	params.Set("scope", "identify")
	params.Set("state", c.CurrentSession.CSRFToken)
	params.Set("redirect_uri", hmnurl.BuildDiscordOAuthCallback())

	td := templateData{
		BaseData:     baseData,
		AuthorizeURL: fmt.Sprintf("https://discord.com/api/oauth2/authorize?%s", params.Encode()),
	}

	if userDiscord != nil {
		u := templates.DiscordUserToTemplate(userDiscord)
		td.DiscordUser = &u
	}

	var res ResponseData
	res.MustWriteTemplate("discordtest.html", td, c.Perf)
	return res
}

func DiscordOAuthCallback(c *RequestContext) ResponseData {
	query := c.Req.URL.Query()

	// Check the state
	state := query.Get("state")
	if state != c.CurrentSession.CSRFToken {
		// CSRF'd!!!!

		// TODO(compression): Should this and the CSRF middleware be pulled out to
		// a separate function?

		c.Logger.Warn().Str("userId", c.CurrentUser.Username).Msg("user failed Discord OAuth state validation - potential attack?")

		err := auth.DeleteSession(c.Context(), c.Conn, c.CurrentSession.ID)
		if err != nil {
			c.Logger.Error().Err(err).Msg("failed to delete session on Discord OAuth state failure")
		}

		res := c.Redirect("/", http.StatusSeeOther)
		res.SetCookie(auth.DeleteSessionCookie)

		return res
	}

	// Check for error values and redirect back to ????
	if query.Get("error") != "" {
		// TODO: actually handle these errors
		return ErrorResponse(http.StatusBadRequest, errors.New(query.Get("error")))
	}

	// Do the actual token exchange and redirect back to ????
	code := query.Get("code")
	res, err := discord.ExchangeOAuthCode(c.Context(), code, hmnurl.BuildDiscordOAuthCallback()) // TODO: Redirect to the right place
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to exchange Discord authorization code"))
	}
	expiry := time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)

	user, err := discord.GetCurrentUserAsOAuth(c.Context(), res.AccessToken)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch Discord user info"))
	}

	// TODO: Add the role on Discord
	err = discord.AddGuildMemberRole(c.Context(), user.ID, config.Config.Discord.MemberRoleID)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to add member role"))
	}

	_, err = c.Conn.Exec(c.Context(),
		`
		INSERT INTO handmade_discorduser (username, discriminator, access_token, refresh_token, avatar, locale, userid, expiry, hmn_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`,
		user.Username,
		user.Discriminator,
		res.AccessToken,
		res.RefreshToken,
		user.Avatar,
		user.Locale,
		user.ID,
		expiry,
		c.CurrentUser.ID,
	)
	if err != nil {
		return ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save new Discord user info"))
	}

	return c.Redirect(hmnurl.BuildDiscordTest(), http.StatusSeeOther)
}

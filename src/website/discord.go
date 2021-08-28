package website

import (
	"errors"
	"net/http"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
)

func DiscordOAuthCallback(c *RequestContext) ResponseData {
	query := c.Req.URL.Query()

	// Check the state
	state := query.Get("state")
	if state != c.CurrentSession.CSRFToken {
		// CSRF'd!!!!

		c.Logger.Warn().Str("userId", c.CurrentUser.Username).Msg("user failed Discord OAuth state validation - potential attack?")

		res := c.Redirect("/", http.StatusSeeOther)
		logoutUser(c, &res)

		return res
	}

	// Check for error values and redirect back to user settings
	if errCode := query.Get("error"); errCode != "" {
		if errCode == "access_denied" {
			// This occurs when the user cancels. Just go back to the profile page.
			return c.Redirect(hmnurl.BuildUserSettings("discord"), http.StatusSeeOther)
		} else {
			return RejectRequest(c, "Failed to authenticate with Discord.")
		}
	}

	// Do the actual token exchange
	code := query.Get("code")
	res, err := discord.ExchangeOAuthCode(c.Context(), code, hmnurl.BuildDiscordOAuthCallback())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to exchange Discord authorization code"))
	}
	expiry := time.Now().Add(time.Duration(res.ExpiresIn) * time.Second)

	user, err := discord.GetCurrentUserAsOAuth(c.Context(), res.AccessToken)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch Discord user info"))
	}

	// Add the role on Discord
	err = discord.AddGuildMemberRole(c.Context(), user.ID, config.Config.Discord.MemberRoleID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to add member role"))
	}

	// Add the user to our database
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
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save new Discord user info"))
	}

	return c.Redirect(hmnurl.BuildUserSettings("discord"), http.StatusSeeOther)
}

func DiscordUnlink(c *RequestContext) ResponseData {
	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	iDiscordUser, err := db.QueryOne(c.Context(), tx, models.DiscordUser{},
		`
		SELECT $columns
		FROM handmade_discorduser
		WHERE hmn_user_id = $1
		`,
		c.CurrentUser.ID,
	)
	if err != nil {
		if errors.Is(err, db.ErrNoMatchingRows) {
			return c.Redirect(hmnurl.BuildUserSettings("discord"), http.StatusSeeOther)
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get Discord user for unlink"))
		}
	}
	discordUser := iDiscordUser.(*models.DiscordUser)

	_, err = tx.Exec(c.Context(),
		`
		DELETE FROM handmade_discorduser
		WHERE id = $1
		`,
		discordUser.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete Discord user"))
	}

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit Discord user delete"))
	}

	err = discord.RemoveGuildMemberRole(c.Context(), discordUser.UserID, config.Config.Discord.MemberRoleID)
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to remove member role on unlink")
	}

	return c.Redirect(hmnurl.BuildUserSettings("discord"), http.StatusSeeOther)
}

func DiscordShowcaseBacklog(c *RequestContext) ResponseData {
	iduser, err := db.QueryOne(c.Context(), c.Conn, models.DiscordUser{},
		`SELECT $columns FROM handmade_discorduser WHERE hmn_user_id = $1`,
		c.CurrentUser.ID,
	)
	if errors.Is(err, db.ErrNoMatchingRows) {
		// Nothing to do
		c.Logger.Warn().Msg("could not do showcase backlog because no discord user exists")
		return c.Redirect(hmnurl.BuildUserProfile(c.CurrentUser.Username), http.StatusSeeOther)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get discord user"))
	}
	duser := iduser.(*models.DiscordUser)

	ok, err := discord.AllowedToCreateMessageSnippet(c.Context(), c.Conn, duser.UserID)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	if !ok {
		// Not allowed to do this, bail out
		c.Logger.Warn().Msg("was not allowed to save user snippets")
		return c.Redirect(hmnurl.BuildUserProfile(c.CurrentUser.Username), http.StatusSeeOther)
	}

	type messageIdQuery struct {
		MessageID string `db:"msg.id"`
	}
	imsgIds, err := db.Query(c.Context(), c.Conn, messageIdQuery{},
		`
		SELECT $columns
		FROM
			handmade_discordmessage AS msg
		WHERE
			msg.user_id = $1
			AND msg.channel_id = $2
		`,
		duser.UserID,
		config.Config.Discord.ShowcaseChannelID,
	)
	msgIds := imsgIds.ToSlice()

	for _, imsgId := range msgIds {
		msgId := imsgId.(*messageIdQuery)
		_, err := discord.CreateMessageSnippet(c.Context(), c.Conn, msgId.MessageID)
		if err != nil {
			c.Logger.Warn().Err(err).Msg("failed to create snippet from showcase backlog")
			continue
		}
	}

	return c.Redirect(hmnurl.BuildUserProfile(c.CurrentUser.Username), http.StatusSeeOther)
}

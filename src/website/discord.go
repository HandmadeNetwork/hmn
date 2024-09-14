package website

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/google/uuid"
)

// This callback handles Discord account linking whether the user is signed in
// or not. In all cases, the end state is that the user is signed into a
// Handmade Network account with a linked Discord account. HMN accounts will be
// created as necessary.
//
// If we initiate OAuth while logged in, we will use the current session's CSRF
// token as the OAuth state. Otherwise, we will generate a new entry in the
// pending_login table with an equivalently random token and use that token for
// the state.
//
// Considerations:
//
// |                       | Already signed in    | Not signed in                 |
// |-----------------------|----------------------|-------------------------------|
// | No matching info      | Link Discord account | Create HMN account            |
// |-----------------------| to current HMN       |-------------------------------|
// | Matching Discord user | account (stealing is | Log into HMN account and link |
// |-----------------------| ok, but make sure    | Discord user to it. (Double-  |
// | One matching email    | any other accounts   | check Discord link settings.) |
// |-----------------------| are unlinked)        |-------------------------------|
// | More than one         |                      | Fail login                    |
// | matching email        |                      |                               |
// |-----------------------|----------------------|-------------------------------|
func DiscordOAuthCallback(c *RequestContext) ResponseData {
	query := c.Req.URL.Query()

	var destinationUrl string

	tx, err := c.Conn.Begin(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to start transaction for Discord OAuth"))
	}
	defer tx.Rollback(c)

	// Check the state, figure out where we're going
	state := query.Get("state")
	if c.CurrentUser == nil {
		// Check the state against all our pending signins - if none is found,
		// then CSRF'd!!!! (or the login just expired)
		pendingLogin, err := db.QueryOne[models.PendingLogin](c, c.Conn,
			`
			SELECT $columns
			FROM pending_login
			WHERE
				id = $1
				AND expires_at > CURRENT_TIMESTAMP
			`,
			state,
		)
		if err == db.NotFound {
			c.Logger.Warn().Str("userId", c.CurrentUser.Username).Msg("user failed Discord OAuth state validation - potential attack?")
			res := c.Redirect("/", http.StatusSeeOther)
			logoutUser(c, &res)
			return res
		} else if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up pending login"))
		}
		destinationUrl = pendingLogin.DestinationUrl

		// Delete the pending login; we're done with it
		_, err = tx.Exec(c, `DELETE FROM pending_login WHERE id = $1`, pendingLogin.ID)
		if err != nil {
			c.Logger.Warn().Str("id", pendingLogin.ID).Err(err).Msg("failed to delete pending login")
		}
	} else {
		// Check the state against the current session - if it does not match,
		// then CSRF'd!!!!
		if state != c.CurrentSession.CSRFToken {
			c.Logger.Warn().Str("userId", c.CurrentUser.Username).Msg("user failed Discord OAuth state validation - potential attack?")
			res := c.Redirect("/", http.StatusSeeOther)
			logoutUser(c, &res)
			return res
		}
		// The only way into OAuth when logged in is when linking your Discord
		// account in settings.
		destinationUrl = hmnurl.BuildUserSettings("discord")
	}

	// Check for error values and redirect back to from whence they came
	if errCode := query.Get("error"); errCode != "" {
		if errCode == "access_denied" {
			// This occurs when the user cancels. Just go back so they can try again.
			var dest string
			if c.CurrentUser == nil {
				// Send 'em back to the login page for another go, with the
				// same destination
				dest = hmnurl.BuildLoginPage(destinationUrl)
			} else {
				dest = hmnurl.BuildUserSettings("discord")
			}
			return c.Redirect(dest, http.StatusSeeOther)
		} else {
			return c.RejectRequest("Failed to authenticate with Discord.")
		}
	}

	// Do the actual token exchange
	code := query.Get("code")
	authRes, err := discord.ExchangeOAuthCode(c, code, hmnurl.BuildDiscordOAuthCallback())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to exchange Discord authorization code"))
	}
	expiry := time.Now().Add(time.Duration(authRes.ExpiresIn) * time.Second)

	user, err := discord.GetCurrentUserAsOAuth(c, authRes.AccessToken)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch Discord user info"))
	}

	hmnMember, err := discord.GetGuildMember(c, config.Config.Discord.GuildID, user.ID)
	if err != nil {
		if err == discord.NotFound {
			// nothing, this is fine
		} else {
			c.Logger.Error().Err(err).Msg("failed to get HMN Discord member for Discord user")
		}
	}

	// Make the necessary updates in our database (see table above)

	// Determine which HMN user to associate this Discord login with. This
	// may not turn anything up, in which case we need to make an account.
	var hmnUser *models.User
	if c.CurrentUser != nil {
		hmnUser = c.CurrentUser
	} else {
		utils.Assert(user.Email, "didn't get an email from Discord! bad scopes?")

		userFromDiscordID, err := db.QueryOne[models.User](c, tx,
			`
			SELECT $columns{hmn_user}
			FROM
				discord_user
				JOIN hmn_user ON discord_user.hmn_user_id = hmn_user.id
			WHERE userid = $1
			`,
			user.ID,
		)
		if err == nil {
			hmnUser = userFromDiscordID
		} else if err == db.NotFound {
			// no problem
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up existing HMN user from Discord OAuth"))
		}

		if hmnUser == nil {
			usersFromDiscordEmail, err := db.Query[models.User](c, tx,
				`
				SELECT $columns
				FROM hmn_user
				WHERE
					LOWER(email) = LOWER($1)
				`,
				user.Email,
			)
			if err == nil {
				if len(usersFromDiscordEmail) > 1 {
					// oh no why don't we have a unique constraint on emails
					return c.RejectRequest("There are multiple Handmade Network accounts with this email address. Please sign into one of them separately.")
				} else if len(usersFromDiscordEmail) == 1 {
					hmnUser = usersFromDiscordEmail[0]
				}
			} else if err == db.NotFound {
				// no problem
			} else {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to look up existing HMN user by email"))
			}
		}
	}

	// Create a new HMN account if no existing account matches
	if hmnUser == nil {
		// Check if an HMN account already has this username. We don't link
		// in this case because usernames can be changed and we don't want
		// account takeovers.
		usernameTaken, err := db.QueryOneScalar[bool](c, tx,
			`SELECT COUNT(*) > 0 FROM hmn_user WHERE LOWER(username) = LOWER($1)`,
			user.Username,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to check if username was taken when logging in with Discord"))
		}
		if usernameTaken {
			return c.RejectRequest(fmt.Sprintf("There is already a Handmade Network account with the username \"%s\".", user.Username))
		}

		var displayName string
		if hmnMember != nil && hmnMember.Nick != nil {
			displayName = *hmnMember.Nick
		}

		var avatarHash *string
		if hmnMember != nil && hmnMember.Avatar != nil {
			avatarHash = hmnMember.Avatar
		} else if user.Avatar != nil {
			avatarHash = user.Avatar
		}

		var avatarAssetID *uuid.UUID
		if avatarHash != nil {
			// Note! Not using the transaction here. Don't want to fail the login due to avatars.
			if avatarAsset, err := saveDiscordAvatar(c, c.Conn, user.ID, *user.Avatar); err == nil {
				avatarAssetID = &avatarAsset.ID
			} else {
				c.Logger.Warn().Err(err).Msg("failed to save Discord avatar")
			}
		}

		newHMNUser, err := db.QueryOne[models.User](c, tx,
			`
			INSERT INTO hmn_user (
				username, name, email, password, avatar_asset_id, date_joined, registration_ip
			) VALUES (
				$1,       $2,   $3,    '',       $4,              $5,          $6
			)
			RETURNING $columns
			`,
			user.Username, displayName, strings.ToLower(user.Email), avatarAssetID, time.Now(), c.GetIP(),
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to create new HMN user for Discord login"))
		}
		hmnUser = newHMNUser
	}

	// Add the Discord user data to our database
	_, err = tx.Exec(c,
		`
		INSERT INTO
		discord_user (username, discriminator, access_token, refresh_token, avatar, locale, userid, expiry, hmn_user_id)
		VALUES       ($1,       $2,            $3,           $4,            $5,     $6,     $7,     $8,     $9)
		ON CONFLICT (userid) DO UPDATE SET
			username = EXCLUDED.username,
			discriminator = EXCLUDED.discriminator,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			avatar = EXCLUDED.avatar,
			locale = EXCLUDED.locale,
			expiry = EXCLUDED.expiry,
			hmn_user_id = EXCLUDED.hmn_user_id
		`,
		user.Username,
		user.Discriminator,
		authRes.AccessToken,
		authRes.RefreshToken,
		user.Avatar,
		user.Locale,
		user.ID,
		expiry,
		hmnUser.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save new Discord user info"))
	}

	// Mark the HMN user as confirmed - Discord is good enough auth for us
	_, err = tx.Exec(c,
		`
		UPDATE hmn_user
		SET status = $1
		WHERE id = $2
		`,
		models.UserStatusApproved,
		hmnUser.ID,
	)
	if err != nil {
		c.Logger.Error().Err(err).Msg("failed to set user status to approved after linking discord account")
		// NOTE(asaf): It's not worth failing the request over this, so we're not returning an error to the user.
	}

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save updates from Discord OAuth"))
	}

	// Add the role on Discord
	if hmnMember != nil {
		err = discord.AddGuildMemberRole(c, user.ID, config.Config.Discord.MemberRoleID)
		if err != nil {
			c.Logger.Error().Err(err).Msg("failed to add member role")
		}
	}

	res := c.Redirect(destinationUrl, http.StatusSeeOther)
	err = loginUser(c, hmnUser, &res)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}
	return res
}

func DiscordUnlink(c *RequestContext) ResponseData {
	tx, err := c.Conn.Begin(c)
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c)

	discordUser, err := db.QueryOne[models.DiscordUser](c, tx,
		`
		SELECT $columns
		FROM discord_user
		WHERE hmn_user_id = $1
		`,
		c.CurrentUser.ID,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			return c.Redirect(hmnurl.BuildUserSettings("discord"), http.StatusSeeOther)
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get Discord user for unlink"))
		}
	}

	_, err = tx.Exec(c,
		`
		DELETE FROM discord_user
		WHERE id = $1
		`,
		discordUser.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete Discord user"))
	}

	err = tx.Commit(c)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to commit Discord user delete"))
	}

	err = discord.RemoveGuildMemberRole(c, discordUser.UserID, config.Config.Discord.MemberRoleID)
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to remove member role on unlink")
	}

	return c.Redirect(hmnurl.BuildUserSettings("discord"), http.StatusSeeOther)
}

func DiscordShowcaseBacklog(c *RequestContext) ResponseData {
	duser, err := db.QueryOne[models.DiscordUser](c, c.Conn,
		`SELECT $columns FROM discord_user WHERE hmn_user_id = $1`,
		c.CurrentUser.ID,
	)
	if errors.Is(err, db.NotFound) {
		// Nothing to do
		c.Logger.Warn().Msg("could not do showcase backlog because no discord user exists")
		return c.Redirect(hmnurl.BuildUserProfile(c.CurrentUser.Username), http.StatusSeeOther)
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to get discord user"))
	}

	msgIDs, err := db.QueryScalar[string](c, c.Conn,
		`
		SELECT msg.id
		FROM
			discord_message AS msg
		WHERE
			msg.user_id = $1
			AND msg.channel_id = $2
		`,
		duser.UserID,
		config.Config.Discord.ShowcaseChannelID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, err)
	}

	for _, msgID := range msgIDs {
		interned, err := discord.FetchInternedMessage(c, c.Conn, msgID)
		if err != nil && !errors.Is(err, db.NotFound) {
			return c.ErrorResponse(http.StatusInternalServerError, err)
		} else if err == nil {
			// NOTE(asaf): Creating snippet even if the checkbox is off because the user asked us to.
			err = discord.HandleSnippetForInternedMessage(c, c.Conn, interned, true, false)
			if err != nil {
				return c.ErrorResponse(http.StatusInternalServerError, err)
			}
		}
	}

	return c.Redirect(hmnurl.BuildUserProfile(c.CurrentUser.Username), http.StatusSeeOther)
}

func saveDiscordAvatar(ctx context.Context, conn db.ConnOrTx, userID, avatarHash string) (*models.Asset, error) {
	const size = 256

	filename := fmt.Sprintf("%s.png", avatarHash)
	url := fmt.Sprintf("https://cdn.discordapp.com/avatars/%s/%s?size=%d", userID, filename, size)

	content, _, err := discord.DownloadDiscordResource(ctx, url)
	if err != nil {
		return nil, oops.New(err, "failed to download Discord avatar")
	}

	asset, err := assets.Create(ctx, conn, assets.CreateInput{
		Content:     content,
		Filename:    filename,
		ContentType: "image/png",

		Width:  size,
		Height: size,
	})
	if err != nil {
		return nil, oops.New(err, "failed to save asset for Discord attachment")
	}

	return asset, nil
}

func DiscordBotDebugPage(c *RequestContext) ResponseData {
	type DiscordBotDebugData struct {
		templates.BaseData
		BotEvents []discord.BotEvent
	}
	botEvents := discord.GetBotEvents()
	var res ResponseData
	res.MustWriteTemplate("discord_bot_debug.html", DiscordBotDebugData{
		BaseData: getBaseData(c, "", nil),

		BotEvents: botEvents,
	}, c.Perf)
	return res
}

package website

import (
	"errors"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"git.handmade.network/hmn/hmn/src/assets"
	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	hmnemail "git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"git.handmade.network/hmn/hmn/src/twitch"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v4"
)

type UserProfileTemplateData struct {
	templates.BaseData
	ProfileUser         templates.User
	ProfileUserLinks    []templates.Link
	ProfileUserProjects []templates.Project
	TimelineItems       []templates.TimelineItem
	OwnProfile          bool
	ShowcaseUrl         string

	CanAddProject bool
	NewProjectUrl string

	AdminSetStatusUrl string
	AdminNukeUrl      string
}

func UserProfile(c *RequestContext) ResponseData {
	username, hasUsername := c.PathParams["username"]

	if !hasUsername || len(strings.TrimSpace(username)) == 0 {
		return FourOhFour(c)
	}

	username = strings.ToLower(username)

	var profileUser *models.User
	if c.CurrentUser != nil && strings.ToLower(c.CurrentUser.Username) == username {
		profileUser = c.CurrentUser
	} else {
		c.Perf.StartBlock("SQL", "Fetch user")
		user, err := db.QueryOne[models.User](c.Context(), c.Conn,
			`
			SELECT $columns{hmn_user}
			FROM
				hmn_user
				LEFT JOIN asset AS hmn_user_avatar ON hmn_user_avatar.id = hmn_user.avatar_asset_id
			WHERE
				LOWER(hmn_user.username) = $1
			`,
			username,
		)
		c.Perf.EndBlock()
		if err != nil {
			if errors.Is(err, db.NotFound) {
				return FourOhFour(c)
			} else {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user: %s", username))
			}
		}
		profileUser = user
	}

	{
		userIsUnapproved := profileUser.Status != models.UserStatusApproved
		canViewUnapprovedUser := c.CurrentUser != nil && (c.CurrentUser.ID == profileUser.ID || c.CurrentUser.IsStaff)
		if userIsUnapproved && !canViewUnapprovedUser {
			return FourOhFour(c)
		}
	}

	c.Perf.StartBlock("SQL", "Fetch user links")
	userLinks, err := db.Query[models.Link](c.Context(), c.Conn,
		`
		SELECT $columns
		FROM
			link as link
		WHERE
			link.user_id = $1
		ORDER BY link.ordering ASC
		`,
		profileUser.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch links for user: %s", username))
	}
	profileUserLinks := make([]templates.Link, 0, len(userLinks))
	for _, l := range userLinks {
		profileUserLinks = append(profileUserLinks, templates.LinkToTemplate(l))
	}
	c.Perf.EndBlock()

	projectsAndStuff, err := hmndata.FetchProjects(c.Context(), c.Conn, c.CurrentUser, hmndata.ProjectsQuery{
		OwnerIDs:      []int{profileUser.ID},
		Lifecycles:    models.AllProjectLifecycles,
		IncludeHidden: true,
		OrderBy:       "all_last_updated DESC",
	})
	templateProjects := make([]templates.Project, 0, len(projectsAndStuff))
	numPersonalProjects := 0
	for _, p := range projectsAndStuff {
		templateProject := templates.ProjectAndStuffToTemplate(&p, hmndata.UrlContextForProject(&p.Project).BuildHomepage(), c.Theme)
		templateProjects = append(templateProjects, templateProject)

		if p.Project.Personal {
			numPersonalProjects++
		}
	}
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetch posts")
	posts, err := hmndata.FetchPosts(c.Context(), c.Conn, c.CurrentUser, hmndata.PostsQuery{
		UserIDs:        []int{profileUser.ID},
		SortDescending: true,
	})
	c.Perf.EndBlock()

	snippets, err := hmndata.FetchSnippets(c.Context(), c.Conn, c.CurrentUser, hmndata.SnippetQuery{
		OwnerIDs: []int{profileUser.ID},
	})
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for user: %s", username))
	}

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	c.Perf.StartBlock("PROFILE", "Construct timeline items")
	timelineItems := make([]templates.TimelineItem, 0, len(posts)+len(snippets))

	for _, post := range posts {
		timelineItems = append(timelineItems, PostToTimelineItem(
			hmndata.UrlContextForProject(&post.Project),
			lineageBuilder,
			&post.Post,
			&post.Thread,
			profileUser,
			c.Theme,
		))
	}

	for _, s := range snippets {
		item := SnippetToTimelineItem(
			&s.Snippet,
			s.Asset,
			s.DiscordMessage,
			s.Tags,
			profileUser,
			c.Theme,
		)
		item.SmallInfo = true
		timelineItems = append(timelineItems, item)
	}

	c.Perf.StartBlock("PROFILE", "Sort timeline")
	sort.Slice(timelineItems, func(i, j int) bool {
		return timelineItems[j].Date.Before(timelineItems[i].Date)
	})
	c.Perf.EndBlock()

	c.Perf.EndBlock()

	templateUser := templates.UserToTemplate(profileUser, c.Theme)

	baseData := getBaseDataAutocrumb(c, templateUser.Name)

	var res ResponseData
	res.MustWriteTemplate("user_profile.html", UserProfileTemplateData{
		BaseData:            baseData,
		ProfileUser:         templateUser,
		ProfileUserLinks:    profileUserLinks,
		ProfileUserProjects: templateProjects,
		TimelineItems:       timelineItems,
		OwnProfile:          (c.CurrentUser != nil && c.CurrentUser.ID == profileUser.ID),
		ShowcaseUrl:         hmnurl.BuildShowcase(),

		CanAddProject: numPersonalProjects < maxPersonalProjects,
		NewProjectUrl: hmnurl.BuildProjectNew(),

		AdminSetStatusUrl: hmnurl.BuildAdminSetUserStatus(),
		AdminNukeUrl:      hmnurl.BuildAdminNukeUser(),
	}, c.Perf)
	return res
}

var UserAvatarMaxFileSize = 1 * 1024 * 1024

func UserSettings(c *RequestContext) ResponseData {
	var res ResponseData

	type UserSettingsTemplateData struct {
		templates.BaseData

		AvatarMaxFileSize int
		DefaultAvatarUrl  string

		User      templates.User
		Email     string // these fields are handled specially on templates.User
		ShowEmail bool
		LinksText string

		SubmitUrl  string
		ContactUrl string

		DiscordUser               *templates.DiscordUser
		DiscordNumUnsavedMessages int
		DiscordAuthorizeUrl       string
		DiscordUnlinkUrl          string
		DiscordShowcaseBacklogUrl string
	}

	links, err := db.Query[models.Link](c.Context(), c.Conn,
		`
		SELECT $columns
		FROM link
		WHERE user_id = $1
		ORDER BY ordering
		`,
		c.CurrentUser.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user links"))
	}

	linksText := LinksToText(links)

	var tduser *templates.DiscordUser
	var numUnsavedMessages int
	duser, err := db.QueryOne[models.DiscordUser](c.Context(), c.Conn,
		`
		SELECT $columns
		FROM discord_user
		WHERE hmn_user_id = $1
		`,
		c.CurrentUser.ID,
	)
	if errors.Is(err, db.NotFound) {
		// this is fine, but don't fetch any more messages
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user's Discord account"))
	} else {
		tmp := templates.DiscordUserToTemplate(duser)
		tduser = &tmp

		numUnsavedMessages, err = db.QueryOneScalar[int](c.Context(), c.Conn,
			`
			SELECT COUNT(*)
			FROM
				discord_message AS msg
				LEFT JOIN discord_message_content AS c ON c.message_id = msg.id
			WHERE
				msg.user_id = $1
				AND msg.channel_id = $2
				AND c.last_content IS NULL
			`,
			duser.UserID,
			config.Config.Discord.ShowcaseChannelID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to check for unsaved user messages"))
		}
	}

	templateUser := templates.UserToTemplate(c.CurrentUser, c.Theme)

	baseData := getBaseDataAutocrumb(c, templateUser.Name)

	res.MustWriteTemplate("user_settings.html", UserSettingsTemplateData{
		BaseData:          baseData,
		AvatarMaxFileSize: UserAvatarMaxFileSize,
		DefaultAvatarUrl:  templates.UserAvatarDefaultUrl(c.Theme),
		User:              templateUser,
		Email:             c.CurrentUser.Email,
		ShowEmail:         c.CurrentUser.ShowEmail,
		LinksText:         linksText,

		SubmitUrl:  hmnurl.BuildUserSettings(""),
		ContactUrl: hmnurl.BuildContactPage(),

		DiscordUser:               tduser,
		DiscordNumUnsavedMessages: numUnsavedMessages,
		DiscordAuthorizeUrl:       discord.GetAuthorizeUrl(c.CurrentSession.CSRFToken),
		DiscordUnlinkUrl:          hmnurl.BuildDiscordUnlink(),
		DiscordShowcaseBacklogUrl: hmnurl.BuildDiscordShowcaseBacklog(),
	}, c.Perf)
	return res
}

func UserSettingsSave(c *RequestContext) ResponseData {
	maxBodySize := int64(UserAvatarMaxFileSize + 2*1024*1024)
	c.Req.Body = http.MaxBytesReader(c.Res, c.Req.Body, maxBodySize)
	err := c.Req.ParseMultipartForm(maxBodySize)
	if err != nil {
		// NOTE(asaf): The error for exceeding the max filesize doesn't have a special type, so we can't easily detect it here.
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to parse form"))
	}

	tx, err := c.Conn.Begin(c.Context())
	if err != nil {
		panic(err)
	}
	defer tx.Rollback(c.Context())

	form, err := c.GetFormValues()
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to parse form on user update")
		return c.Redirect(hmnurl.BuildUserSettings(""), http.StatusSeeOther)
	}

	name := form.Get("realname")

	email := form.Get("email")
	if !hmnemail.IsEmail(email) {
		return RejectRequest(c, "Your email was not valid.")
	}

	showEmail := form.Get("showemail") != ""
	darkTheme := form.Get("darktheme") != ""

	blurb := form.Get("shortbio")
	signature := form.Get("signature")
	bio := form.Get("longbio")

	discordShowcaseAuto := form.Get("discord-showcase-auto") != ""
	discordDeleteSnippetOnMessageDelete := form.Get("discord-snippet-keep") == ""

	_, err = tx.Exec(c.Context(),
		`
		UPDATE hmn_user
		SET
			name = $2,
			email = $3,
			showemail = $4,
			darktheme = $5,
			blurb = $6,
			signature = $7,
			bio = $8,
			discord_save_showcase = $9,
			discord_delete_snippet_on_message_delete = $10
		WHERE
			id = $1
		`,
		c.CurrentUser.ID,
		name,
		email,
		showEmail,
		darkTheme,
		blurb,
		signature,
		bio,
		discordShowcaseAuto,
		discordDeleteSnippetOnMessageDelete,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update user"))
	}

	// Process links
	twitchLoginsPreChange, preErr := hmndata.FetchTwitchLoginsForUserOrProject(c.Context(), tx, &c.CurrentUser.ID, nil)
	linksText := form.Get("links")
	links := ParseLinks(linksText)
	_, err = tx.Exec(c.Context(), `DELETE FROM link WHERE user_id = $1`, c.CurrentUser.ID)
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to delete old links")
	} else {
		for i, link := range links {
			_, err := tx.Exec(c.Context(),
				`
				INSERT INTO link (name, url, ordering, user_id)
				VALUES ($1, $2, $3, $4)
				`,
				link.Name,
				link.Url,
				i,
				c.CurrentUser.ID,
			)
			if err != nil {
				c.Logger.Warn().Err(err).Msg("failed to insert new link")
				continue
			}
		}
	}
	twitchLoginsPostChange, postErr := hmndata.FetchTwitchLoginsForUserOrProject(c.Context(), tx, &c.CurrentUser.ID, nil)
	if preErr == nil && postErr == nil {
		twitch.UserOrProjectLinksUpdated(twitchLoginsPreChange, twitchLoginsPostChange)
	}

	// Update password
	oldPassword := form.Get("old_password")
	newPassword := form.Get("new_password1")
	newPasswordConfirmation := form.Get("new_password2")
	if oldPassword != "" && newPassword != "" {
		errorRes := updatePassword(c, tx, oldPassword, newPassword, newPasswordConfirmation)
		if errorRes != nil {
			return *errorRes
		}
	}

	// Update avatar
	newAvatar, err := GetFormImage(c, "avatar")
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to read image from form"))
	}
	var avatarUUID *uuid.UUID
	if newAvatar.Exists {
		avatarAsset, err := assets.Create(c.Context(), tx, assets.CreateInput{
			Content:     newAvatar.Content,
			Filename:    newAvatar.Filename,
			ContentType: newAvatar.Mime,
			UploaderID:  &c.CurrentUser.ID,
			Width:       newAvatar.Width,
			Height:      newAvatar.Height,
		})
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to upload avatar"))
		}
		avatarUUID = &avatarAsset.ID
	}
	if newAvatar.Exists || newAvatar.Remove {
		_, err := tx.Exec(c.Context(),
			`
			UPDATE hmn_user
			SET
				avatar_asset_id = $2
			WHERE
				id = $1
			`,
			c.CurrentUser.ID,
			avatarUUID,
		)
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update user's avatar"))
		}
	}

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save user settings"))
	}

	res := c.Redirect(hmnurl.BuildUserSettings(""), http.StatusSeeOther)
	res.AddFutureNotice("success", "User profile updated.")

	return res
}

func UserProfileAdminSetStatus(c *RequestContext) ResponseData {
	c.Req.ParseForm()

	userIdStr := c.Req.Form.Get("user_id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		return RejectRequest(c, "No user id provided")
	}

	status := c.Req.Form.Get("status")
	var desiredStatus models.UserStatus
	switch status {
	case "inactive":
		desiredStatus = models.UserStatusInactive
	case "confirmed":
		desiredStatus = models.UserStatusConfirmed
	case "approved":
		desiredStatus = models.UserStatusApproved
	case "banned":
		desiredStatus = models.UserStatusBanned
	default:
		return RejectRequest(c, "No legal user status provided")
	}

	_, err = c.Conn.Exec(c.Context(),
		`
		UPDATE hmn_user
		SET status = $1
		WHERE id = $2
		`,
		desiredStatus,
		userId,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update user status"))
	}
	if desiredStatus == models.UserStatusBanned {
		err = auth.DeleteSessionForUser(c.Context(), c.Conn, c.Req.Form.Get("username"))
		if err != nil {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to log out user"))
		}
	}
	res := c.Redirect(hmnurl.BuildUserProfile(c.Req.Form.Get("username")), http.StatusSeeOther)
	res.AddFutureNotice("success", "Successfully set status")
	return res
}

func UserProfileAdminNuke(c *RequestContext) ResponseData {
	c.Req.ParseForm()
	userIdStr := c.Req.Form.Get("user_id")
	userId, err := strconv.Atoi(userIdStr)
	if err != nil {
		return RejectRequest(c, "No user id provided")
	}

	err = deleteAllPostsForUser(c.Context(), c.Conn, userId)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to delete user posts"))
	}
	res := c.Redirect(hmnurl.BuildUserProfile(c.Req.Form.Get("username")), http.StatusSeeOther)
	res.AddFutureNotice("success", "Successfully nuked user")
	return res
}

func updatePassword(c *RequestContext, tx pgx.Tx, old, new, confirm string) *ResponseData {
	if new != confirm {
		res := RejectRequest(c, "Your password and password confirmation did not match.")
		return &res
	}

	oldHashedPassword, err := auth.ParsePasswordString(c.CurrentUser.Password)
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to parse user's password string")
		return nil
	}

	ok, err := auth.CheckPassword(old, oldHashedPassword)
	if err != nil {
		res := c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to check user's password"))
		return &res
	}

	if !ok {
		res := RejectRequest(c, "The old password you provided was not correct.")
		return &res
	}

	newHashedPassword := auth.HashPassword(new)
	err = auth.UpdatePassword(c.Context(), tx, c.CurrentUser.Username, newHashedPassword)
	if err != nil {
		res := c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to update password"))
		return &res
	}

	return nil
}

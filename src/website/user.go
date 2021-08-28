package website

import (
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"git.handmade.network/hmn/hmn/src/auth"
	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	hmnemail "git.handmade.network/hmn/hmn/src/email"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/templates"
	"github.com/jackc/pgx/v4"
)

type UserProfileTemplateData struct {
	templates.BaseData
	ProfileUser         templates.User
	ProfileUserLinks    []templates.Link
	ProfileUserProjects []templates.Project
	TimelineItems       []templates.TimelineItem
	NumForums           int
	NumBlogs            int
	NumSnippets         int
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
		userResult, err := db.QueryOne(c.Context(), c.Conn, models.User{},
			`
			SELECT $columns
			FROM
				auth_user
			WHERE
				LOWER(auth_user.username) = $1
			`,
			username,
		)
		c.Perf.EndBlock()
		if err != nil {
			if errors.Is(err, db.ErrNoMatchingRows) {
				return FourOhFour(c)
			} else {
				return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user: %s", username))
			}
		}
		profileUser = userResult.(*models.User)
	}
	c.Perf.StartBlock("SQL", "Fetch user links")
	type userLinkQuery struct {
		UserLink models.Link `db:"link"`
	}
	userLinkQueryResult, err := db.Query(c.Context(), c.Conn, userLinkQuery{},
		`
		SELECT $columns
		FROM
			handmade_links as link
		WHERE
			link.user_id = $1
		ORDER BY link.ordering ASC
		`,
		profileUser.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch links for user: %s", username))
	}
	userLinksSlice := userLinkQueryResult.ToSlice()
	profileUserLinks := make([]templates.Link, 0, len(userLinksSlice))
	for _, l := range userLinksSlice {
		profileUserLinks = append(profileUserLinks, templates.LinkToTemplate(&l.(*userLinkQuery).UserLink))
	}
	c.Perf.EndBlock()

	type projectQuery struct {
		Project models.Project `db:"project"`
	}
	c.Perf.StartBlock("SQL", "Fetch projects")
	projectQueryResult, err := db.Query(c.Context(), c.Conn, projectQuery{},
		`
		SELECT $columns
		FROM
			handmade_project AS project
			INNER JOIN handmade_user_projects AS uproj ON uproj.project_id = project.id
		WHERE
			uproj.user_id = $1
			AND ($2 OR (project.flags = 0 AND project.lifecycle = ANY ($3)))
		`,
		profileUser.ID,
		(c.CurrentUser != nil && (profileUser == c.CurrentUser || c.CurrentUser.IsStaff)),
		models.VisibleProjectLifecycles,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch projects for user: %s", username))
	}
	projectQuerySlice := projectQueryResult.ToSlice()
	templateProjects := make([]templates.Project, 0, len(projectQuerySlice))
	for _, projectRow := range projectQuerySlice {
		projectData := projectRow.(*projectQuery)
		templateProjects = append(templateProjects, templates.ProjectToTemplate(&projectData.Project, c.Theme))
	}
	c.Perf.EndBlock()

	type postQuery struct {
		Post    models.Post    `db:"post"`
		Thread  models.Thread  `db:"thread"`
		Project models.Project `db:"project"`
	}
	c.Perf.StartBlock("SQL", "Fetch posts")
	postQueryResult, err := db.Query(c.Context(), c.Conn, postQuery{},
		`
		SELECT $columns
		FROM
			handmade_post AS post
			INNER JOIN handmade_thread AS thread ON thread.id = post.thread_id
			INNER JOIN handmade_project AS project ON project.id = post.project_id
		WHERE
			post.author_id = $1
			AND project.lifecycle = ANY ($2)
		`,
		profileUser.ID,
		models.VisibleProjectLifecycles,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch posts for user: %s", username))
	}
	postQuerySlice := postQueryResult.ToSlice()
	c.Perf.EndBlock()

	type snippetQuery struct {
		Snippet        models.Snippet         `db:"snippet"`
		Asset          *models.Asset          `db:"asset"`
		DiscordMessage *models.DiscordMessage `db:"discord_message"`
	}
	c.Perf.StartBlock("SQL", "Fetch snippets")
	snippetQueryResult, err := db.Query(c.Context(), c.Conn, snippetQuery{},
		`
		SELECT $columns
		FROM
			handmade_snippet AS snippet
			LEFT JOIN handmade_asset AS asset ON asset.id = snippet.asset_id
			LEFT JOIN handmade_discordmessage AS discord_message ON discord_message.id = snippet.discord_message_id
		WHERE
			snippet.owner_id = $1
		`,
		profileUser.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch snippets for user: %s", username))
	}
	snippetQuerySlice := snippetQueryResult.ToSlice()
	c.Perf.EndBlock()

	c.Perf.StartBlock("SQL", "Fetch subforum tree")
	subforumTree := models.GetFullSubforumTree(c.Context(), c.Conn)
	lineageBuilder := models.MakeSubforumLineageBuilder(subforumTree)
	c.Perf.EndBlock()

	c.Perf.StartBlock("PROFILE", "Construct timeline items")
	timelineItems := make([]templates.TimelineItem, 0, len(postQuerySlice)+len(snippetQuerySlice))
	numForums := 0
	numBlogs := 0
	numSnippets := len(snippetQuerySlice)

	for _, postRow := range postQuerySlice {
		postData := postRow.(*postQuery)
		timelineItem := PostToTimelineItem(
			lineageBuilder,
			&postData.Post,
			&postData.Thread,
			&postData.Project,
			profileUser,
			c.Theme,
		)
		switch timelineItem.Type {
		case templates.TimelineTypeForumThread, templates.TimelineTypeForumReply:
			numForums += 1
		case templates.TimelineTypeBlogPost, templates.TimelineTypeBlogComment:
			numBlogs += 1
		}
		if timelineItem.Type != templates.TimelineTypeUnknown {
			timelineItems = append(timelineItems, timelineItem)
		} else {
			c.Logger.Warn().Int("post ID", postData.Post.ID).Msg("Unknown timeline item type for post")
		}
	}

	for _, snippetRow := range snippetQuerySlice {
		snippetData := snippetRow.(*snippetQuery)
		timelineItem := SnippetToTimelineItem(
			&snippetData.Snippet,
			snippetData.Asset,
			snippetData.DiscordMessage,
			profileUser,
			c.Theme,
		)
		timelineItems = append(timelineItems, timelineItem)
	}

	c.Perf.StartBlock("PROFILE", "Sort timeline")
	sort.Slice(timelineItems, func(i, j int) bool {
		return timelineItems[j].Date.Before(timelineItems[i].Date)
	})
	c.Perf.EndBlock()

	c.Perf.EndBlock()

	templateUser := templates.UserToTemplate(profileUser, c.Theme)

	baseData := getBaseData(c)
	baseData.Title = templateUser.Name

	var res ResponseData
	res.MustWriteTemplate("user_profile.html", UserProfileTemplateData{
		BaseData:            baseData,
		ProfileUser:         templateUser,
		ProfileUserLinks:    profileUserLinks,
		ProfileUserProjects: templateProjects,
		TimelineItems:       timelineItems,
		NumForums:           numForums,
		NumBlogs:            numBlogs,
		NumSnippets:         numSnippets,
	}, c.Perf)
	return res
}

func UserSettings(c *RequestContext) ResponseData {
	var res ResponseData

	type UserSettingsTemplateData struct {
		templates.BaseData

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

	ilinks, err := db.Query(c.Context(), c.Conn, models.Link{},
		`
		SELECT $columns
		FROM handmade_links
		WHERE user_id = $1
		ORDER BY ordering
		`,
		c.CurrentUser.ID,
	)
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user links"))
	}
	links := ilinks.ToSlice()

	linksText := ""
	for _, ilink := range links {
		link := ilink.(*models.Link)
		linksText += fmt.Sprintf("%s %s\n", link.URL, link.Name)
	}

	var tduser *templates.DiscordUser
	var numUnsavedMessages int
	iduser, err := db.QueryOne(c.Context(), c.Conn, models.DiscordUser{},
		`
		SELECT $columns
		FROM handmade_discorduser
		WHERE hmn_user_id = $1
		`,
		c.CurrentUser.ID,
	)
	if errors.Is(err, db.ErrNoMatchingRows) {
		// this is fine, but don't fetch any more messages
	} else if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to fetch user's Discord account"))
	} else {
		duser := iduser.(*models.DiscordUser)
		tmp := templates.DiscordUserToTemplate(duser)
		tduser = &tmp

		numUnsavedMessages, err = db.QueryInt(c.Context(), c.Conn,
			`
			SELECT COUNT(*)
			FROM
				handmade_discordmessage AS msg
				LEFT JOIN handmade_discordmessagecontent AS c ON c.message_id = msg.id
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

	baseData := getBaseData(c)
	baseData.Title = templateUser.Name

	res.MustWriteTemplate("user_settings.html", UserSettingsTemplateData{
		BaseData:  baseData,
		User:      templateUser,
		Email:     c.CurrentUser.Email,
		ShowEmail: c.CurrentUser.ShowEmail,
		LinksText: linksText,

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
		UPDATE auth_user
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
	linksText := form.Get("links")
	links := strings.Split(linksText, "\n")
	_, err = tx.Exec(c.Context(), `DELETE FROM handmade_links WHERE user_id = $1`, c.CurrentUser.ID)
	if err != nil {
		c.Logger.Warn().Err(err).Msg("failed to delete old links")
	} else {
		for i, link := range links {
			link = strings.TrimSpace(link)
			linkParts := strings.SplitN(link, " ", 2)
			url := strings.TrimSpace(linkParts[0])
			name := ""
			if len(linkParts) > 1 {
				name = strings.TrimSpace(linkParts[1])
			}

			if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
				continue
			}

			_, err := tx.Exec(c.Context(),
				`
				INSERT INTO handmade_links (name, url, ordering, user_id)
				VALUES ($1, $2, $3, $4)
				`,
				name,
				url,
				i,
				c.CurrentUser.ID,
			)
			if err != nil {
				c.Logger.Warn().Err(err).Msg("failed to insert new link")
				continue
			}
		}
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
	_, err = SaveImageFile(c, tx, "avatar", 1*1024*1024, fmt.Sprintf("members/avatars/%s", c.CurrentUser.Username))
	if err != nil {
		var rejectErr RejectRequestError
		if errors.As(err, &rejectErr) {
			return RejectRequest(c, rejectErr.Error())
		} else {
			return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save new avatar"))
		}
	}

	// TODO: Success message

	err = tx.Commit(c.Context())
	if err != nil {
		return c.ErrorResponse(http.StatusInternalServerError, oops.New(err, "failed to save user settings"))
	}

	return c.Redirect(hmnurl.BuildUserSettings(""), http.StatusSeeOther)
}

// TODO: Rework this to use that RejectRequestError thing
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

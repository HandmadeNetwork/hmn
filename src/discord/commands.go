package discord

import (
	"context"
	"errors"
	"fmt"

	"git.handmade.network/hmn/hmn/src/hmndata"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
)

// Slash command names and options
const SlashCommandProfile = "profile"
const ProfileOptionUser = "user"

// User command names
const UserCommandProfile = "HMN Profile"

func (bot *botInstance) createApplicationCommands(ctx context.Context) {
	doOrWarn := func(err error) {
		if err == nil {
			logging.ExtractLogger(ctx).Info().Msg("Created Discord application command")
		} else {
			logging.ExtractLogger(ctx).Warn().Err(err).Msg("Failed to create Discord application command")
		}
	}

	doOrWarn(CreateGuildApplicationCommand(ctx, CreateGuildApplicationCommandRequest{
		Type:        ApplicationCommandTypeChatInput,
		Name:        SlashCommandProfile,
		Description: "Get a link to a user's Handmade Network profile",
		Options: []ApplicationCommandOption{
			{
				Type:        ApplicationCommandOptionTypeUser,
				Name:        ProfileOptionUser,
				Description: "The Discord user to look up on Handmade Network",
				Required:    true,
			},
		},
	}))
	doOrWarn(CreateGuildApplicationCommand(ctx, CreateGuildApplicationCommandRequest{
		Type: ApplicationCommandTypeUser,
		Name: UserCommandProfile,
	}))
}

func (bot *botInstance) doInteraction(ctx context.Context, i *Interaction) {
	defer func() {
		if recovered := recover(); recovered != nil {
			logger := logging.ExtractLogger(ctx).Error()
			if err, ok := recovered.(error); ok {
				logger = logger.Err(err)
			} else {
				logger = logger.Interface("recovered", recovered)
			}
			logger.Msg("panic when handling Discord interaction")
		}
	}()

	switch i.Data.Name {
	case SlashCommandProfile:
		userOpt := mustGetInteractionOption(i.Data.Options, ProfileOptionUser)
		userID := userOpt.Value.(string)
		bot.handleProfileCommand(ctx, i, userID)
	case UserCommandProfile:
		bot.handleProfileCommand(ctx, i, i.Data.TargetID)
	default:
		logging.ExtractLogger(ctx).Warn().Str("name", i.Data.Name).Msg("didn't recognize Discord interaction name")
	}
}

func (bot *botInstance) handleProfileCommand(ctx context.Context, i *Interaction, userID string) {
	member := i.Data.Resolved.Members[userID]

	if member.User.IsBot {
		err := CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
			Type: InteractionCallbackTypeChannelMessageWithSource,
			Data: &InteractionCallbackData{
				Content: "<a:confusedparrot:865957487026765864>",
				Flags:   FlagEphemeral,
			},
		})
		if err != nil {
			logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send profile response")
		}
		return
	}

	hmnUser, err := db.QueryOne[models.User](ctx, bot.dbConn,
		`
		SELECT $columns{hmn_user}
		FROM
			discord_user AS duser
			JOIN hmn_user ON duser.hmn_user_id = hmn_user.id
		WHERE
			duser.userid = $1
			AND hmn_user.status = $2
		`,
		userID,
		models.UserStatusApproved,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			err = CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
				Type: InteractionCallbackTypeChannelMessageWithSource,
				Data: &InteractionCallbackData{
					Content: fmt.Sprintf("<@%s> hasn't linked a Handmade Network profile.", member.User.ID),
					Flags:   FlagEphemeral,
				},
			})
			if err != nil {
				logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send profile response")
			}
		} else {
			logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to look up user profile")
		}
		return
	}

	projectsAndStuff, err := hmndata.FetchProjects(ctx, bot.dbConn, nil, hmndata.ProjectsQuery{
		OwnerIDs: []int{hmnUser.ID},
	})
	if err != nil {
		logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to fetch user projects")
	}

	url := hmnurl.BuildUserProfile(hmnUser.Username)
	msg := fmt.Sprintf("<@%s>'s profile can be viewed at %s.", member.User.ID, url)
	if len(projectsAndStuff) > 0 {
		projectNoun := "projects"
		if len(projectsAndStuff) == 1 {
			projectNoun = "project"
		}
		msg += fmt.Sprintf(" They have %d %s:\n", len(projectsAndStuff), projectNoun)

		for _, p := range projectsAndStuff {
			msg += fmt.Sprintf("- %s: %s\n", p.Project.Name, hmndata.UrlContextForProject(&p.Project).BuildHomepage())
		}
	}

	err = CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
		Type: InteractionCallbackTypeChannelMessageWithSource,
		Data: &InteractionCallbackData{
			Content: msg,
			Flags:   FlagEphemeral,
		},
	})
	if err != nil {
		logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send profile response")
	}
}

func getInteractionOption(opts []ApplicationCommandInteractionDataOption, name string) (ApplicationCommandInteractionDataOption, bool) {
	for _, opt := range opts {
		if opt.Name == name {
			return opt, true
		}
	}

	return ApplicationCommandInteractionDataOption{}, false
}

func mustGetInteractionOption(opts []ApplicationCommandInteractionDataOption, name string) ApplicationCommandInteractionDataOption {
	opt, ok := getInteractionOption(opts, name)
	if !ok {
		panic(fmt.Errorf("failed to get interaction option with name '%s'", name))
	}
	return opt
}

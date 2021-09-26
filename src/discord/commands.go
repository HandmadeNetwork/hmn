package discord

import (
	"context"
	"errors"
	"fmt"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
)

const CommandNameProfile = "profile"

const ProfileCommandOptionUser = "user"

func (bot *botInstance) createApplicationCommands(ctx context.Context) {
	doOrWarn := func(err error) {
		if err == nil {
			logging.ExtractLogger(ctx).Info().Msg("Created Discord application command")
		} else {
			logging.ExtractLogger(ctx).Warn().Err(err).Msg("Failed to create Discord application command")
		}
	}

	doOrWarn(CreateGuildApplicationCommand(ctx, CreateGuildApplicationCommandRequest{
		Name:        CommandNameProfile,
		Description: "Get a link to a user's Handmade Network profile",
		Options: []ApplicationCommandOption{
			{
				Type:        ApplicationCommandOptionTypeUser,
				Name:        ProfileCommandOptionUser,
				Description: "The Discord user to look up on Handmade Network",
				Required:    true,
			},
		},
		Type: ApplicationCommandTypeChatInput,
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
	case CommandNameProfile:
		bot.handleProfileCommand(ctx, i)
	default:
		logging.ExtractLogger(ctx).Warn().Str("name", i.Data.Name).Msg("didn't recognize Discord interaction name")
	}
}

func (bot *botInstance) handleProfileCommand(ctx context.Context, i *Interaction) {
	userOpt := mustGetInteractionOption(i.Data.Options, ProfileCommandOptionUser)
	userID := userOpt.Value.(string)
	member := i.Data.Resolved.Members[userID]

	if userID == config.Config.Discord.BotUserID {
		err := CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
			Type: InteractionCallbackTypeChannelMessageWithSource,
			Data: &InteractionCallbackData{
				Content: "<a:confusedparrot:891814826484064336>",
			},
		})
		if err != nil {
			logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send profile response")
		}
		return
	}

	type profileResult struct {
		HMNUser models.User `db:"auth_user"`
	}
	ires, err := db.QueryOne(ctx, bot.dbConn, profileResult{},
		`
		SELECT $columns
		FROM
			handmade_discorduser AS duser
			JOIN auth_user ON duser.hmn_user_id = auth_user.id
		WHERE
			duser.userid = $1
		`,
		userID,
	)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			err = CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
				Type: InteractionCallbackTypeChannelMessageWithSource,
				Data: &InteractionCallbackData{
					Content: fmt.Sprintf("%s hasn't linked a Handmade Network profile.", member.DisplayName()),
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
	res := ires.(*profileResult)

	url := hmnurl.BuildUserProfile(res.HMNUser.Username)
	err = CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
		Type: InteractionCallbackTypeChannelMessageWithSource,
		Data: &InteractionCallbackData{
			Content: fmt.Sprintf("%s's profile can be viewed at %s.", member.DisplayName(), url),
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

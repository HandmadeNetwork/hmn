package discord

import (
	"context"
	"errors"
	"fmt"

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
	res := ires.(*profileResult)

	url := hmnurl.BuildUserProfile(res.HMNUser.Username)
	err = CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
		Type: InteractionCallbackTypeChannelMessageWithSource,
		Data: &InteractionCallbackData{
			Content: fmt.Sprintf("<@%s>'s profile can be viewed at %s.", member.User.ID, url),
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

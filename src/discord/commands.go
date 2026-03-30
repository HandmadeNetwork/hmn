package discord

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/utils"

	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
)

// Slash command names and options
const SlashCommandProfile = "profile"
const ProfileOptionUser = "user"

const SlashCommandManifesto = "manifesto"
const SlashCommandHMHReplay = "hmhreplay"
const HMHReplayOptionToggle = "toggle"

const SlashCommandJoinJam = "joinjam"

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

	doOrWarn(CreateGuildApplicationCommand(ctx, CreateGuildApplicationCommandRequest{
		Type:        ApplicationCommandTypeChatInput,
		Name:        SlashCommandManifesto,
		Description: "Read the Handmade manifesto",
	}))

	doOrWarn(CreateGuildApplicationCommand(ctx, CreateGuildApplicationCommandRequest{
		Type:         ApplicationCommandTypeChatInput,
		Name:         SlashCommandHMHReplay,
		Description:  "Join the Handmade Hero Replay",
		DMPermission: utils.P(false),
		Options: []ApplicationCommandOption{
			{
				Type:        ApplicationCommandOptionTypeBoolean,
				Name:        HMHReplayOptionToggle,
				Description: "Add or remove the HMH replay role",
				Required:    false,
			},
		},
	}))

	doOrWarn(CreateGuildApplicationCommand(ctx, CreateGuildApplicationCommandRequest{
		Type:         ApplicationCommandTypeChatInput,
		Name:         SlashCommandJoinJam,
		Description:  "Join an upcoming jam",
		DMPermission: utils.P(false),
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
	case SlashCommandManifesto:
		err := CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
			Type: InteractionCallbackTypeChannelMessageWithSource,
			Data: &InteractionCallbackData{
				Content: "Read the Handmade manifesto at https://handmade.network/manifesto",
				Flags:   FlagEphemeral,
			},
		})
		if err != nil {
			logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send manifesto response")
		}
	case SlashCommandHMHReplay:
		join := true
		toggleOpt, ok := getInteractionOption(i.Data.Options, HMHReplayOptionToggle)
		if ok {
			join = toggleOpt.Value.(bool)
		}

		if join {
			logging.ExtractLogger(ctx).Debug().Interface("interaction", i).Msg("Got hmh replay")
			err := AddGuildMemberRole(ctx, i.Member.User.ID, config.Config.Discord.HMHReplayRoleID)
			if err != nil {
				err = sendEphemeralMessageForInteraction(ctx, i, "We failed to set the role. Please inform an admin.")
				if err != nil {
					logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send hmh replay add failure response")
				}
			} else {
				err = sendEphemeralMessageForInteraction(ctx, i, "You're in!")
				if err != nil {
					logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send hmh replay add response")
				}
			}
		} else {
			err := RemoveGuildMemberRole(ctx, i.Member.User.ID, config.Config.Discord.HMHReplayRoleID)
			if err != nil {
				err = sendEphemeralMessageForInteraction(ctx, i, "We failed to remove the role. Please inform an admin.")
				if err != nil {
					logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send hmh replay remove failure response")
				}
			} else {
				err = sendEphemeralMessageForInteraction(ctx, i, "You're out!")
				if err != nil {
					logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send hmh replay remove response")
				}
			}
		}

	case SlashCommandJoinJam:
		log := logging.ExtractLogger(ctx).With().Str("interaction", i.ID).Logger()
		log.Debug().Interface("interaction", i).Msg("Got /joinjam")

		conn := db.NewConn()

		// Look up jam, respond if there is none
		if !hmndata.LatestJam.WithinGrace(time.Now(), hmndata.JamProjectCreateGracePeriod, 0) {
			sendEphemeralMessageForInteraction(ctx, i, "You are not able to join a jam right now.")
			return
		}

		// Look up linked HMN user, respond with settings link if none found
		_, err := hmndata.FetchUser(ctx, conn, nil, hmndata.UsersQuery{
			DiscordUserIDs: []string{i.Member.User.ID},
		})
		if err == db.NotFound {
			settingsUrl := hmnurl.BuildUserSettings("discord")
			loginUrl := hmnurl.BuildLoginPage(settingsUrl, "joinjam")
			sendEphemeralMessageForInteraction(ctx, i, fmt.Sprintf(
				"You must link a Handmade Network account to your Discord account. [Sign up](%s) and link your Discord account in settings, then re-run this command.",
				loginUrl,
			))
			return
		} else if err != nil {
			log.Error().Err(err).Msg("failed to look up Discord user")
			sendEphemeralMessageForInteraction(ctx, i, "Failed to look up your linked HMN account. Please contact an admin.")
			return
		}

		errOccurred := false

		// Give user the Discord role
		roleID := hmndata.LatestJam.DiscordRoleIDs[config.Config.Env]
		utils.Assert(roleID)
		err = AddGuildMemberRole(ctx, i.Member.User.ID, roleID)
		if err != nil {
			log.Error().Err(err).Msg("failed to give user the jam role")
			errOccurred = true
		}

		// Ping user in #jam
		utils.Assert(config.Config.Discord.JamChannelID)
		err = SendMessages(ctx, conn, MessageToSend{
			ChannelID: config.Config.Discord.JamChannelID,
			Req: CreateMessageRequest{
				Content: fmt.Sprintf("<@%s> has joined the jam!", i.Member.User.ID),
			},
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to ping the user in the #jam channel")
			errOccurred = true
		}

		if errOccurred {
			sendEphemeralMessageForInteraction(ctx, i, "Something went wrong while signing you up for the jam. Please contact an admin.")
		} else {
			sendEphemeralMessageForInteraction(ctx, i, "You are now signed up for the jam! The next step is to create a project to act as your submission. See the pinned message in #jam for more info.")
		}

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
	var msg strings.Builder
	msg.WriteString(fmt.Sprintf("<@%s>'s profile can be viewed at %s.", member.User.ID, url))
	if len(projectsAndStuff) > 0 {
		projectNoun := "projects"
		if len(projectsAndStuff) == 1 {
			projectNoun = "project"
		}
		msg.WriteString(fmt.Sprintf(" They have %d %s:\n", len(projectsAndStuff), projectNoun))

		for _, p := range projectsAndStuff {
			msg.WriteString(fmt.Sprintf("- %s: %s\n", p.Project.Name, hmndata.UrlContextForProject(&p.Project).BuildHomepage()))
		}
	}

	err = CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
		Type: InteractionCallbackTypeChannelMessageWithSource,
		Data: &InteractionCallbackData{
			Content: msg.String(),
			Flags:   FlagEphemeral,
		},
	})
	if err != nil {
		logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to send profile response")
	}
}

func sendEphemeralMessageForInteraction(ctx context.Context, i *Interaction, content string) error {
	err := CreateInteractionResponse(ctx, i.ID, i.Token, InteractionResponse{
		Type: InteractionCallbackTypeChannelMessageWithSource,
		Data: &InteractionCallbackData{
			Content: content,
			Flags:   FlagEphemeral,
		},
	})

	return err
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

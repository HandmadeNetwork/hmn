package website

import (
	"context"
	"fmt"
	"slices"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/discord"
	"git.handmade.network/hmn/hmn/src/hmndata"
	"git.handmade.network/hmn/hmn/src/hmnurl"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func NagUsersToCreateJamProjectsJob(dbConn *pgxpool.Pool) *jobs.Job {
	job := jobs.New("jam project creation nag bot")
	log := job.Logger.With().Str("job", "jam nag bot").Logger()

	go func() {
		defer func() {
			log.Info().Msg("Shutting down jam project creation nag bot")
			job.Finish()
		}()
		log.Info().Msg("Running jam project creation nag bot")

		t := time.NewTimer(time.Minute)
		lastTime := time.Now()

		for {
			select {
			case now := <-t.C:
				for _, jam := range hmndata.AllJams {
					if utils.TimeIsBetween(lastTime, jam.NagTime, now) {
						nags, err := NagUsersToCreateJamProjects(job.Ctx, dbConn, &jam)
						if err != nil {
							log.Error().Err(err).Msg("Failed to nag people about the jam")
						} else {
							log.Info().Strs("hmnusers", nags).Msg("Nagged users about the jam")
						}
					}
				}
				lastTime = now
			case <-job.Canceled():
				return
			}
		}
	}()

	return job
}

func NagUsersToCreateJamProjects(ctx context.Context, conn db.ConnOrTx, jam *hmndata.Jam) ([]string, error) {
	utils.Assert(jam.DiscordRoleIDs != nil)
	jamRole := jam.DiscordRoleIDs[config.Config.Env]
	utils.Assert(jamRole)

	allMembers, err := discord.ListGuildMembers(ctx, config.Config.Discord.GuildID)
	if err != nil {
		return nil, oops.New(err, "failed to list all guild members for nag")
	}

	// Get all Discord users with the jam role
	var discordUserIDsWithRole []string
	for _, member := range allMembers {
		utils.Assert(member.User, "members from ListGuildMembers should always have a user")
		if slices.Contains(member.Roles, jamRole) {
			discordUserIDsWithRole = append(discordUserIDsWithRole, member.User.ID)
		}
	}

	// Look up all HMN users that are linked to those users. (It should be ok to use a user ID of nil
	// here because everyone should be auto-approved when linking a Discord account.)
	linkedHMNUsersList, err := hmndata.FetchUsers(ctx, conn, nil, hmndata.UsersQuery{
		DiscordUserIDs: discordUserIDsWithRole,
	})
	if err != nil {
		return nil, oops.New(err, "failed to look up linked HMN users for jam nag")
	}

	// Collect all the HMN users into a map keyed by Discord user ID, and a list of IDs for further queries
	linkedHMNUserIDs := make([]int, len(linkedHMNUsersList))
	linkedHMNUsers := make(map[string]*models.User, len(linkedHMNUsersList))
	for _, u := range linkedHMNUsersList {
		linkedHMNUserIDs = append(linkedHMNUserIDs, u.ID)
		linkedHMNUsers[u.DiscordUser.UserID] = u
	}

	// Look up all jam projects for all those HMN users
	jamProjects, err := hmndata.FetchProjects(ctx, conn, nil, hmndata.ProjectsQuery{
		OwnerIDs: linkedHMNUserIDs,
		JamSlugs: []string{jam.Slug},

		// Include hidden projects so people don't get nagged about them
		IncludeHidden: true,
		ShowJamHidden: true,
	})
	if err != nil {
		return nil, oops.New(err, "failed to look up jam projects for nag")
	}

	// Finally, for all users, check if they have linked an account and created a project.
	var nags []string
	for _, discordUserID := range discordUserIDsWithRole {
		settingsUrl := hmnurl.BuildUserSettings("discord")
		loginUrl := hmnurl.BuildLoginPage(settingsUrl, "joinjam")
		projectCreateUrl := hmnurl.BuildProjectNewJam()

		hmnUser, ok := linkedHMNUsers[discordUserID]
		if !ok {
			msg := fmt.Sprintf(
				`You recently signed up for the %s, but you have not yet linked a Handmade Network account. There's a couple things still to do:

1. [Sign up](<%s>) for a Handmade Network account and link your Discord account in settings. (Pro tip: Sign in with Discord to do both in one step.)
2. [Create a Handmade Network project](<%s>) to act as your submission.

Please take care of those so you can participate!
`,
				jam.Name, loginUrl, projectCreateUrl,
			)

			nags = append(nags, discordUserID)
			discord.SendDM(ctx, conn, discordUserID, msg)
			continue
		}

		hasProject := false
	findProject:
		for _, proj := range jamProjects {
			for _, owner := range proj.Owners {
				if owner.ID == hmnUser.ID {
					hasProject = true
					break findProject
				}
			}
		}
		if !hasProject {
			msg := fmt.Sprintf(
				`You recently signed up for the %s, but you have not yet [created a Handmade Network project](<%s>). Please create one before the jam starts so you can start sharing progress updates right away!`,
				jam.Name, projectCreateUrl,
			)

			nags = append(nags, hmnUser.Username)
			discord.SendDM(ctx, conn, discordUserID, msg)
		}
	}

	return nags, nil
}

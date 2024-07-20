package discord

import (
	"context"
	"errors"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/jackc/pgx/v5/pgxpool"
)

func RunHistoryWatcher(ctx context.Context, dbConn *pgxpool.Pool) jobs.Job {
	log := logging.ExtractLogger(ctx).With().Str("discord goroutine", "history watcher").Logger()
	ctx = logging.AttachLoggerToContext(&log, ctx)

	if config.Config.Discord.BotToken == "" {
		log.Warn().Msg("No Discord bot token was provided, so the Discord history bot cannot run.")
		return jobs.Noop()
	}

	job := jobs.New()
	go func() {
		defer func() {
			log.Debug().Msg("shut down Discord history watcher")
			job.Done()
		}()

		newUserTicker := time.NewTicker(5 * time.Second)

		backfillFirstRun := make(chan struct{}, 1)
		backfillFirstRun <- struct{}{}
		backfillTicker := time.NewTicker(1 * time.Hour)

		lastBackfillTime := time.Now().Add(-3 * time.Hour)

		runBackfill := func() {
			log.Info().Msg("Running backfill")
			// Run a backfill to patch up places where the Discord bot missed (does create snippets)
			now := time.Now()
			done := Scrape(ctx, dbConn,
				config.Config.Discord.ShowcaseChannelID,
				lastBackfillTime,
				true,
			)
			if done {
				lastBackfillTime = now
			}
		}

		for {
			done, err := func() (done bool, err error) {
				defer utils.RecoverPanicAsError(&err)
				select {
				case <-ctx.Done():
					return true, nil
				case <-newUserTicker.C:
					// Get content for messages when a user links their account (but do not create snippets)
					fetchMissingContent(ctx, dbConn)
				case <-backfillFirstRun:
					runBackfill()
				case <-backfillTicker.C:
					runBackfill()
				}
				return false, nil
			}()
			if err != nil {
				log.Error().Err(err).Msg("Panicked in RunHistoryWatcher")
			} else if done {
				return
			}
		}
	}()

	return job
}

func fetchMissingContent(ctx context.Context, dbConn *pgxpool.Pool) {
	log := logging.ExtractLogger(ctx)

	messagesWithoutContent, err := db.Query[models.DiscordMessage](ctx, dbConn,
		`
		SELECT $columns{msg}
		FROM
			discord_message AS msg
			JOIN discord_user AS duser ON msg.user_id = duser.userid -- only fetch messages for linked discord users
			LEFT JOIN discord_message_content AS c ON c.message_id = msg.id
		WHERE
			c.last_content IS NULL
			AND msg.guild_id = $1
		ORDER BY msg.sent_at DESC
		`,
		config.Config.Discord.GuildID,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check for messages without content")
		return
	}

	if len(messagesWithoutContent) > 0 {
		log.Info().Msgf("There are %d Discord messages without content, fetching their content now...", len(messagesWithoutContent))
	msgloop:
		for _, msg := range messagesWithoutContent {
			select {
			case <-ctx.Done():
				log.Info().Msg("Scrape was canceled")
				break msgloop
			default:
			}

			discordMsg, err := GetChannelMessage(ctx, msg.ChannelID, msg.ID)
			if errors.Is(err, NotFound) {
				// This message has apparently been deleted; delete it from our database
				interned, err := FetchInternedMessage(ctx, dbConn, msg.ID)
				if err != nil {
					if !errors.Is(err, db.NotFound) {
						log.Error().Str("Message ID", msg.ID).Msg("couldn't find interned message")
					} else {
						log.Error().Err(err).Msg("failed to fetch interned message")
					}
					continue
				}
				err = DeleteInternedMessage(ctx, dbConn, interned)
				if err != nil {
					log.Error().Err(err).Msg("failed to delete missing message")
					continue
				}
				log.Info().Str("msg id", msg.ID).Msg("deleted missing Discord message")
				continue
			} else if err != nil {
				log.Error().Err(err).Msg("failed to get message")
				continue
			}

			log.Info().Str("msg", discordMsg.ShortString()).Msg("fetched message for content")

			err = HandleInternedMessage(ctx, dbConn, discordMsg, false, false, false)
			if err != nil {
				log.Error().Err(err).Msg("failed to save content for message")
				continue
			}
		}
		log.Info().Msgf("Done fetching missing content")
	}
}

func Scrape(ctx context.Context, dbConn *pgxpool.Pool, channelID string, earliestMessageTime time.Time, createSnippets bool) bool {
	log := logging.ExtractLogger(ctx)

	log.Info().Msg("Starting scrape")
	defer log.Info().Msg("Done with scrape!")

	before := ""
	for {
		msgs, err := GetChannelMessages(ctx, channelID, GetChannelMessagesInput{
			Limit:  100,
			Before: before,
		})
		if err != nil {
			logging.Error().Err(err).Msg("failed to get messages while scraping")
			return false
		}

		if len(msgs) == 0 {
			logging.Debug().Msg("out of messages, stopping scrape")
			return true
		}

		for _, msg := range msgs {
			select {
			case <-ctx.Done():
				log.Info().Msg("Scrape was canceled")
				return false
			default:
			}

			log.Info().Str("msg", msg.ShortString()).Msg("")

			if !earliestMessageTime.IsZero() && msg.Time().Before(earliestMessageTime) {
				logging.ExtractLogger(ctx).Info().Time("earliest", earliestMessageTime).Msg("Saw a message before the specified earliest time; exiting")
				return true
			}

			msg.Backfilled = true
			err := HandleIncomingMessage(ctx, dbConn, &msg, createSnippets)

			if err != nil {
				errLog := logging.ExtractLogger(ctx).Error()
				errLog.Err(err).Msg("failed to process Discord message")
			}

			before = msg.ID
		}
	}
}

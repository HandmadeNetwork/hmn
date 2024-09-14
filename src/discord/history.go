package discord

import (
	"context"
	"errors"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/jobs"
	"git.handmade.network/hmn/hmn/src/logging"
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
			done := ScrapeRecents(ctx, dbConn,
				config.Config.Discord.ShowcaseChannelID,
				lastBackfillTime,
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

	messagesWithoutContent, err := db.Query[InternedMessage](ctx, dbConn,
		`
		SELECT $columns
		FROM
			discord_message AS message
			JOIN discord_user AS duser ON duser.userid = message.user_id -- only fetch messages for linked discord users
			LEFT JOIN discord_message_content AS content ON content.message_id = message.id
			LEFT JOIN hmn_user AS hmnuser ON hmnuser.id = duser.hmn_user_id
		WHERE
			content.last_content IS NULL
		ORDER BY message.sent_at DESC
		`,
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

			discordMsg, err := GetChannelMessage(ctx, msg.Message.ChannelID, msg.Message.ID)
			if errors.Is(err, NotFound) {
				log.Info().Str("ID", msg.Message.ID).Msg("message deleted on discord")
				err = DeleteInternedMessage(ctx, dbConn, msg)
			} else if err != nil {
				log.Error().Err(err).Msg("failed to get message")
				continue
			} else {
				log.Info().Str("msg", discordMsg.ShortString()).Msg("fetched message for content")

				err = HandleInternedMessage(ctx, dbConn, discordMsg, false, false, false)
				if err != nil {
					log.Error().Err(err).Msg("failed to save content for message")
					continue
				}
			}

		}
		log.Info().Msgf("Done fetching missing content")
	}
}

// NOTE(asaf): Behaves like we're receiving the messages from the gateway. Should only be used to catch up.
func ScrapeRecents(ctx context.Context, dbConn *pgxpool.Pool, channelID string, earliestMessageTime time.Time) bool {
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
			err := HandleIncomingMessage(ctx, dbConn, &msg, false)

			if err != nil {
				errLog := logging.ExtractLogger(ctx).Error()
				errLog.Err(err).Msg("failed to process Discord message")
			}

			before = msg.ID
		}
	}
}

func ScrapeAll(ctx context.Context, dbConn *pgxpool.Pool, channelID string, earliestMessageTime time.Time) bool {
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
			tags := parseTags(msg.Content)
			interned, err := MaybeInternMessage(ctx, dbConn, &msg, tags)
			if interned && err == nil {
				err = HandleInternedMessage(ctx, dbConn, &msg, false, false, false)
			}

			if err != nil {
				errLog := logging.ExtractLogger(ctx).Error()
				errLog.Err(err).Msg("failed to process Discord message")
			}

			before = msg.ID
		}
	}
}

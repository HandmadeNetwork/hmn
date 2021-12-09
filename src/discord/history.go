package discord

import (
	"context"
	"errors"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

func RunHistoryWatcher(ctx context.Context, dbConn *pgxpool.Pool) <-chan struct{} {
	log := logging.ExtractLogger(ctx).With().Str("discord goroutine", "history watcher").Logger()
	ctx = logging.AttachLoggerToContext(&log, ctx)

	if config.Config.Discord.BotToken == "" {
		log.Warn().Msg("No Discord bot token was provided, so the Discord history bot cannot run.")
		done := make(chan struct{}, 1)
		done <- struct{}{}
		return done
	}

	done := make(chan struct{})
	go func() {
		defer func() {
			log.Debug().Msg("shut down Discord history watcher")
			done <- struct{}{}
		}()

		backfillInterval := 1 * time.Hour

		newUserTicker := time.NewTicker(5 * time.Second)
		backfillTicker := time.NewTicker(backfillInterval)

		lastBackfillTime := time.Now().Add(-backfillInterval)
		for {
			select {
			case <-ctx.Done():
				return
			case <-newUserTicker.C:
				// Get content for messages when a user links their account (but do not create snippets)
				fetchMissingContent(ctx, dbConn)
			case <-backfillTicker.C:
				// Run a backfill to patch up places where the Discord bot missed (does create snippets)
				Scrape(ctx, dbConn,
					config.Config.Discord.ShowcaseChannelID,
					lastBackfillTime,
					true,
				)
			}
		}
	}()

	return done
}

func fetchMissingContent(ctx context.Context, dbConn *pgxpool.Pool) {
	log := logging.ExtractLogger(ctx)

	type query struct {
		Message models.DiscordMessage `db:"msg"`
	}
	result, err := db.Query(ctx, dbConn, query{},
		`
		SELECT $columns
		FROM
			handmade_discordmessage AS msg
			JOIN handmade_discorduser AS duser ON msg.user_id = duser.userid -- only fetch messages for linked discord users
			LEFT JOIN handmade_discordmessagecontent AS c ON c.message_id = msg.id
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
	imessagesWithoutContent := result.ToSlice()

	if len(imessagesWithoutContent) > 0 {
		log.Info().Msgf("There are %d Discord messages without content, fetching their content now...", len(imessagesWithoutContent))
	msgloop:
		for _, imsg := range imessagesWithoutContent {
			select {
			case <-ctx.Done():
				log.Info().Msg("Scrape was canceled")
				break msgloop
			default:
			}

			msg := imsg.(*query).Message

			discordMsg, err := GetChannelMessage(ctx, msg.ChannelID, msg.ID)
			if errors.Is(err, NotFound) {
				// This message has apparently been deleted; delete it from our database
				_, err = dbConn.Exec(ctx,
					`
					DELETE FROM handmade_discordmessage
					WHERE id = $1
					`,
					msg.ID,
				)
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

			err = handleHistoryMessage(ctx, dbConn, discordMsg, false)
			if err != nil {
				log.Error().Err(err).Msg("failed to save content for message")
				continue
			}
		}
		log.Info().Msgf("Done fetching missing content")
	}
}

func Scrape(ctx context.Context, dbConn *pgxpool.Pool, channelID string, earliestMessageTime time.Time, createSnippets bool) {
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
			return
		}

		if len(msgs) == 0 {
			logging.Debug().Msg("out of messages, stopping scrape")
			return
		}

		for _, msg := range msgs {
			select {
			case <-ctx.Done():
				log.Info().Msg("Scrape was canceled")
				return
			default:
			}

			log.Info().Str("msg", msg.ShortString()).Msg("")

			if !earliestMessageTime.IsZero() && msg.Time().Before(earliestMessageTime) {
				logging.ExtractLogger(ctx).Info().Time("earliest", earliestMessageTime).Msg("Saw a message before the specified earliest time; exiting")
				return
			}

			err := handleHistoryMessage(ctx, dbConn, &msg, createSnippets)
			if err != nil {
				errLog := logging.ExtractLogger(ctx).Error()
				if errors.Is(err, errNotEnoughInfo) {
					errLog = logging.ExtractLogger(ctx).Warn()
				}
				errLog.Err(err).Msg("failed to process Discord message")
			}

			before = msg.ID
		}
	}
}

func handleHistoryMessage(ctx context.Context, dbConn *pgxpool.Pool, msg *Message, createSnippets bool) error {
	var tx pgx.Tx
	for {
		var err error
		tx, err = dbConn.Begin(ctx)
		if err != nil {
			logging.ExtractLogger(ctx).Warn().Err(err).Msg("failed to start transaction for message")
			time.Sleep(1 * time.Second)
			continue
		}
		break
	}

	newMsg, err := SaveMessageAndContents(ctx, tx, msg)
	if err != nil {
		return err
	}
	if createSnippets {
		if doSnippet, err := AllowedToCreateMessageSnippet(ctx, tx, newMsg.UserID); doSnippet && err == nil {
			_, err := CreateMessageSnippet(ctx, tx, newMsg.UserID, msg.ID)
			if err != nil {
				return err
			}
		} else if err != nil {
			return err
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return err
	}

	return nil
}

package discord

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"net"
	"runtime"
	"sync"
	"time"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/db"
	"git.handmade.network/hmn/hmn/src/logging"
	"git.handmade.network/hmn/hmn/src/models"
	"git.handmade.network/hmn/hmn/src/oops"
	"git.handmade.network/hmn/hmn/src/utils"
	"github.com/gorilla/websocket"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/jpillora/backoff"
)

func RunDiscordBot(ctx context.Context, dbConn *pgxpool.Pool) <-chan struct{} {
	log := logging.ExtractLogger(ctx).With().Str("module", "discord").Logger()
	ctx = logging.AttachLoggerToContext(&log, ctx)

	if config.Config.Discord.BotToken == "" {
		log.Warn().Msg("No Discord bot token was provided, so the Discord bot cannot run.")
		done := make(chan struct{}, 1)
		done <- struct{}{}
		return done
	}

	done := make(chan struct{})
	go func() {
		defer func() {
			log.Debug().Msg("shut down Discord bot")
			done <- struct{}{}
		}()

		boff := backoff.Backoff{
			Min: 1 * time.Second,
			Max: 5 * time.Minute,
		}

		for {
			select {
			case <-ctx.Done():
				return
			default:
			}

			func() {
				log.Info().Msg("Connecting to the Discord gateway")
				bot := newBotInstance(dbConn)
				err := bot.Run(ctx)
				if err != nil {
					dur := boff.Duration()
					log.Error().
						Err(err).
						Dur("retrying after", dur).
						Msg("failed to run Discord bot")

					timer := time.NewTimer(dur)
					select {
					case <-ctx.Done():
					case <-timer.C:
					}

					return
				}

				select {
				case <-ctx.Done():
					return
				default:
				}

				// This delay satisfies the 1 to 5 second delay Discord
				// wants on reconnects, and seems fine to do every time.
				delay := time.Duration(int64(time.Second) + rand.Int63n(int64(time.Second*4)))
				log.Info().Dur("delay", delay).Msg("Reconnecting to Discord")
				time.Sleep(delay)

				boff.Reset()
			}()
		}
	}()
	return done
}

var outgoingMessagesReady = make(chan struct{}, 1)

type botInstance struct {
	conn   *websocket.Conn
	dbConn *pgxpool.Pool

	heartbeatIntervalMs int
	forceHeartbeat      chan struct{}

	/*
	   Every time we send a heartbeat, we set this variable to false.
	   Whenever we ack a heartbeat, we set this variable to true.
	   If we try to send a heartbeat but the previous one was not
	   acked, then we close the connection and try to reconnect.
	*/
	didAckHeartbeat bool

	/*
		All goroutines should call this when they exit, to ensure that
		the other goroutines shut down as well.
	*/
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func newBotInstance(dbConn *pgxpool.Pool) *botInstance {
	return &botInstance{
		dbConn:          dbConn,
		forceHeartbeat:  make(chan struct{}),
		didAckHeartbeat: true,
	}
}

/*
Runs a bot instance to completion. It will start up a gateway connection and return when the
connection is closed. It only returns an error when something unexpected occurs; if so, you should
do exponential backoff before reconnecting. Otherwise you can reconnect right away.
*/
func (bot *botInstance) Run(ctx context.Context) (err error) {
	defer utils.RecoverPanicAsError(&err)

	ctx, bot.cancel = context.WithCancel(ctx)
	defer bot.cancel()

	err = bot.connect(ctx)
	if err != nil {
		return oops.New(err, "failed to connect to Discord gateway")
	}
	defer bot.conn.Close()

	bot.wg.Add(1)
	go bot.doSender(ctx)

	// Wait for child goroutines to exit (they will do so when context is canceled). This ensures
	// that nothing is in the middle of sending. Then close the connection, so that this goroutine
	// can finish as well.
	go func() {
		bot.wg.Wait()
		bot.conn.Close()
	}()

	for {
		msg, err := bot.receiveGatewayMessage(ctx)
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				// If the connection is closed, that's our cue to shut down the bot. Any errors
				// related to the closure will have been logged elsewhere anyway.
				return nil
			} else {
				// NOTE(ben): I don't know what events we might get in the future that we might
				// want to handle gracefully (like above). Keep an eye out.
				return oops.New(err, "failed to receive message from the gateway")
			}
		}

		// Update the sequence number in the db
		if msg.SequenceNumber != nil {
			_, err = bot.dbConn.Exec(ctx, `UPDATE discord_session SET sequence_number = $1`, *msg.SequenceNumber)
			if err != nil {
				return oops.New(err, "failed to save latest sequence number")
			}
		}

		switch msg.Opcode {
		case OpcodeDispatch:
			// Just a normal event
			err := bot.processEventMsg(ctx, msg)
			if err != nil {
				return oops.New(err, "failed to process gateway event")
			}
		case OpcodeHeartbeat:
			bot.forceHeartbeat <- struct{}{}
		case OpcodeHeartbeatACK:
			bot.didAckHeartbeat = true
		case OpcodeReconnect:
			logging.ExtractLogger(ctx).Info().Msg("Discord asked us to reconnect to the gateway")
			return nil
		case OpcodeInvalidSession:
			// We tried to resume but the session was invalid.
			// Delete the session and reconnect from scratch again.
			_, err := bot.dbConn.Exec(ctx, `DELETE FROM discord_session`)
			if err != nil {
				return oops.New(err, "failed to delete invalid session")
			}
			return nil
		}
	}
}

/*
The connection process in short:
- Gateway sends Hello, asking the client to heartbeat on some interval
- Client sends Identify and starts heartbeat process
- Gateway sends Ready, client is now connected to gateway

Or, if we have an existing session:
- Gateway sends Hello, asking the client to heartbeat on some interval
- Client sends Resume and starts heartbeat process
- Gateway sends all missed events followed by a RESUMED event, or an Invalid Session if the
  session is ded

Note that some events probably won't be received until the Guild Create message is received.

It's a little annoying to handle resumes since we want to handle the missed messages as if we were
receiving them in real time. But we're kind of in a different state from from when we're normally
receiving messages, because we are expecting a RESUMED event at the end, and the first message we
receive might be an Invalid Session. So, unfortunately, we just have to handle the Invalid Session
and RESUMED messages in our main message receiving loop instead of here.

(Discord could have prevented this if they send a "Resume ACK" message before replaying events.
That way, we could receive exactly one message after sending Resume, either a Resume ACK or an
Invalid Session, and from there it would be crystal clear what to do. Alas!)
*/
func (bot *botInstance) connect(ctx context.Context) error {
	res, err := GetGatewayBot(ctx)
	if err != nil {
		return oops.New(err, "failed to get gateway URL")
	}

	conn, _, err := websocket.DefaultDialer.Dial(fmt.Sprintf("%s/?v=9&encoding=json", res.URL), nil)
	if err != nil {
		return oops.New(err, "failed to connect to the Discord gateway")
	}
	bot.conn = conn

	helloMessage, err := bot.receiveGatewayMessage(ctx)
	if err != nil {
		return oops.New(err, "failed to read Hello message")
	}
	if helloMessage.Opcode != OpcodeHello {
		return oops.New(nil, "expected a Hello (opcode %d), but got opcode %d", OpcodeHello, helloMessage.Opcode)
	}
	helloData := HelloFromMap(helloMessage.Data)
	bot.heartbeatIntervalMs = helloData.HeartbeatIntervalMs

	// Now that the gateway has said hello, we need to establish a new session, either resuming
	// an old one or starting a new one.

	shouldResume := true
	session, err := db.QueryOne[models.DiscordSession](ctx, bot.dbConn, `SELECT $columns FROM discord_session`)
	if err != nil {
		if errors.Is(err, db.NotFound) {
			// No session yet! Just identify and get on with it
			shouldResume = false
		} else {
			return oops.New(err, "failed to get current session from database")
		}
	}

	if shouldResume {
		// Reconnect to the previous session
		err := bot.sendGatewayMessage(ctx, GatewayMessage{
			Opcode: OpcodeResume,
			Data: Resume{
				Token:          config.Config.Discord.BotToken,
				SessionID:      session.ID,
				SequenceNumber: session.SequenceNumber,
			},
		})
		if err != nil {
			return oops.New(err, "failed to send Resume message")
		}

		return nil
	} else {
		// Start a new session
		err := bot.sendGatewayMessage(ctx, GatewayMessage{
			Opcode: OpcodeIdentify,
			Data: Identify{
				Token: config.Config.Discord.BotToken,
				Properties: IdentifyConnectionProperties{
					OS:      runtime.GOOS,
					Browser: BotName,
					Device:  BotName,
				},
				Intents: IntentGuilds | IntentGuildMessages,
			},
		})
		if err != nil {
			return oops.New(err, "failed to send Identify message")
		}

		readyMessage, err := bot.receiveGatewayMessage(ctx)
		if err != nil {
			return oops.New(err, "failed to read Ready message")
		}
		if readyMessage.Opcode != OpcodeDispatch {
			return oops.New(err, "expected a READY event, but got a message with opcode %d", readyMessage.Opcode)
		}
		if *readyMessage.EventName != "READY" {
			return oops.New(err, "expected a READY event, but got a %s event", *readyMessage.EventName)
		}
		readyData := ReadyFromMap(readyMessage.Data)

		_, err = bot.dbConn.Exec(ctx,
			`
			INSERT INTO discord_session (session_id, sequence_number)
				VALUES ($1, $2)
			ON CONFLICT (pk) DO UPDATE
				SET session_id = $1, sequence_number = $2
			`,
			readyData.SessionID,
			*readyMessage.SequenceNumber,
		)
		if err != nil {
			return oops.New(err, "failed to save new bot session in the database")
		}
	}

	return nil
}

/*
Sends outgoing gateway messages and channel messages. Handles heartbeats. This function should be
run as its own goroutine.
*/
func (bot *botInstance) doSender(ctx context.Context) {
	defer bot.wg.Done()
	defer bot.cancel()

	log := logging.ExtractLogger(ctx).With().Str("discord goroutine", "sender").Logger()
	ctx = logging.AttachLoggerToContext(&log, ctx)

	defer log.Info().Msg("shutting down Discord sender")

	/*
		The first heartbeat is supposed to occur at a random time within
		the first heartbeat interval.

		https://discord.com/developers/docs/topics/gateway#heartbeating
	*/
	dur := time.Duration(bot.heartbeatIntervalMs) * time.Millisecond
	firstDelay := time.NewTimer(time.Duration(rand.Int63n(int64(dur))))
	heartbeatTicker := &time.Ticker{} // this will start never ticking, and get initialized after the first heartbeat

	// Returns false if the heartbeat failed
	sendHeartbeat := func() bool {
		if !bot.didAckHeartbeat {
			log.Error().Msg("did not receive a heartbeat ACK in between heartbeats")
			return false
		}
		bot.didAckHeartbeat = false

		latestSequenceNumber, err := db.QueryOneScalar[int](ctx, bot.dbConn, `SELECT sequence_number FROM discord_session`)
		if err != nil {
			log.Error().Err(err).Msg("failed to fetch latest sequence number from the db")
			return false
		}

		err = bot.sendGatewayMessage(ctx, GatewayMessage{
			Opcode: OpcodeHeartbeat,
			Data:   latestSequenceNumber,
		})
		if err != nil {
			log.Error().Err(err).Msg("failed to send heartbeat")
			return false
		}

		return true
	}

	/*
		Start a goroutine to fetch outgoing messages from the db. We do this in a separate goroutine
		to ensure that issues talking to the database don't prevent us from sending heartbeats.
	*/
	messages := make(chan *models.DiscordOutgoingMessage)
	bot.wg.Add(1)
	go func(ctx context.Context) {
		defer bot.wg.Done()
		defer bot.cancel()

		log := logging.ExtractLogger(ctx).With().Str("discord goroutine", "sender db reader").Logger()
		ctx = logging.AttachLoggerToContext(&log, ctx)

		defer log.Info().Msg("stopping db reader")

		// We will poll the database just in case the notification mechanism doesn't work.
		ticker := time.NewTicker(time.Second * 5)

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			case <-outgoingMessagesReady:
			}

			func() {
				tx, err := bot.dbConn.Begin(ctx)
				if err != nil {
					log.Error().Err(err).Msg("failed to start transaction")
					return
				}
				defer tx.Rollback(ctx)

				msgs, err := db.Query[models.DiscordOutgoingMessage](ctx, tx, `
					SELECT $columns
					FROM discord_outgoing_message
					ORDER BY id ASC
				`)
				if err != nil {
					log.Error().Err(err).Msg("failed to fetch outgoing Discord messages")
					return
				}

				for _, msg := range msgs {
					if time.Now().After(msg.ExpiresAt) {
						continue
					}
					messages <- msg
				}

				/*
					NOTE(ben): Doing this in a transaction means that we will only delete the
					messages that we originally fetched. At least, as long as the database's
					isolation level is Read Committed, which is the default.

					https://www.postgresql.org/docs/current/transaction-iso.html
				*/
				_, err = tx.Exec(ctx, `DELETE FROM discord_outgoing_message`)
				if err != nil {
					log.Error().Err(err).Msg("failed to delete outgoing messages")
					return
				}

				err = tx.Commit(ctx)
				if err != nil {
					log.Error().Err(err).Msg("failed to read and delete outgoing messages")
					return
				}

				if len(msgs) > 0 {
					log.Debug().Int("num messages", len(msgs)).Msg("Sent and deleted outgoing messages")
				}
			}()
		}
	}(ctx)

	/*
		Whenever we want to send a gateway message, we must receive a value from
		this channel first. A goroutine continuously fills the channel at a rate
		that respects Discord's gateway rate limit.

		Don't use this for heartbeats; heartbeats should go out immediately.
		Don't forget that the server can request a heartbeat at any time.

		See the docs for more details. The capacity of this channel is chosen to
		always leave us overhead for heartbeats and other shenanigans.

		https://discord.com/developers/docs/topics/gateway#rate-limiting
	*/
	rateLimiter := make(chan struct{}, 100)
	go func() {
		for {
			rateLimiter <- struct{}{}
			time.Sleep(500 * time.Millisecond)
		}
	}()
	/*
		NOTE(ben): This rate limiter is actually not used right now
		because we're not actually sending any meaningful gateway
		messages. But in the future, if we end up sending presence
		updates or other gateway commands, we need to make sure to
		put this limiter on all of those outgoing commands.
	*/

	for {
		select {
		case <-ctx.Done():
			return
		case <-firstDelay.C:
			if ok := sendHeartbeat(); !ok {
				return
			}
			heartbeatTicker = time.NewTicker(dur)
		case <-heartbeatTicker.C:
			if ok := sendHeartbeat(); !ok {
				return
			}
		case <-bot.forceHeartbeat:
			if ok := sendHeartbeat(); !ok {
				return
			}
			heartbeatTicker.Reset(dur)
		case msg := <-messages:
			_, err := CreateMessage(ctx, msg.ChannelID, msg.PayloadJSON)
			if err != nil {
				log.Error().Err(err).Msg("failed to send Discord message")
			}
		}
	}
}

func (bot *botInstance) receiveGatewayMessage(ctx context.Context) (*GatewayMessage, error) {
	_, msgBytes, err := bot.conn.ReadMessage()
	if err != nil {
		return nil, err
	}

	var msg GatewayMessage
	err = json.Unmarshal(msgBytes, &msg)
	if err != nil {
		return nil, oops.New(err, "failed to unmarshal Discord gateway message")
	}

	logging.ExtractLogger(ctx).Debug().Interface("msg", msg).Msg("received gateway message")

	return &msg, nil
}

func (bot *botInstance) sendGatewayMessage(ctx context.Context, msg GatewayMessage) error {
	logging.ExtractLogger(ctx).Debug().Interface("msg", msg).Msg("sending gateway message")
	return bot.conn.WriteMessage(websocket.TextMessage, msg.ToJSON())
}

/*
Processes a single event message from Discord. If this returns an error, it means something has
really gone wrong, bad enough that the connection should be shut down. Otherwise it will just log
any errors that occur.
*/
func (bot *botInstance) processEventMsg(ctx context.Context, msg *GatewayMessage) error {
	if msg.Opcode != OpcodeDispatch {
		panic(fmt.Sprintf("processEventMsg must only be used on Dispatch messages (opcode %d). Validate this before you call this function.", OpcodeDispatch))
	}

	switch *msg.EventName {
	case "RESUMED":
		// Nothing to do, but at least we can log something
		logging.ExtractLogger(ctx).Info().Msg("Finished resuming gateway session")

		bot.createApplicationCommands(ctx)
	case "MESSAGE_CREATE":
		newMessage := *MessageFromMap(msg.Data, "")

		err := bot.messageCreateOrUpdate(ctx, &newMessage)
		if err != nil {
			return oops.New(err, "error on new message")
		}
	case "MESSAGE_UPDATE":
		newMessage := *MessageFromMap(msg.Data, "")

		err := bot.messageCreateOrUpdate(ctx, &newMessage)
		if err != nil {
			return oops.New(err, "error on updated message")
		}
	case "MESSAGE_DELETE":
		bot.messageDelete(ctx, MessageDeleteFromMap(msg.Data))
	case "MESSAGE_BULK_DELETE":
		bulkDelete := MessageBulkDeleteFromMap(msg.Data)
		for _, id := range bulkDelete.IDs {
			bot.messageDelete(ctx, MessageDelete{
				ID:        id,
				ChannelID: bulkDelete.ChannelID,
				GuildID:   bulkDelete.GuildID,
			})
		}
	case "GUILD_CREATE":
		guild := *GuildFromMap(msg.Data, "")
		if guild.ID != config.Config.Discord.GuildID {
			break
		}

		bot.createApplicationCommands(ctx)
	case "INTERACTION_CREATE":
		go bot.doInteraction(ctx, InteractionFromMap(msg.Data, ""))
	}

	return nil
}

// Only return an error if we want to restart the bot.
func (bot *botInstance) messageCreateOrUpdate(ctx context.Context, msg *Message) error {
	if msg.OriginalHasFields("author") && msg.Author.ID == config.Config.Discord.BotUserID {
		// Don't process your own messages
		return nil
	}

	err := HandleIncomingMessage(ctx, bot.dbConn, msg, true)
	if err != nil {
		logging.ExtractLogger(ctx).Error().Err(err).Msg("failed to handle incoming message")
	}

	// NOTE(asaf): Since any error from HandleIncomingMessage is an internal error and not a discord
	//             error, we only want to log it and not restart the bot. So we're not returning the error.
	return nil
}

func (bot *botInstance) messageDelete(ctx context.Context, msgDelete MessageDelete) {
	log := logging.ExtractLogger(ctx)

	interned, err := FetchInternedMessage(ctx, bot.dbConn, msgDelete.ID)
	if err != nil {
		if !errors.Is(err, db.NotFound) {
			log.Error().Err(err).Msg("failed to fetch interned message")
		}
		return
	}
	err = DeleteInternedMessage(ctx, bot.dbConn, interned)
	if err != nil {
		log.Error().Err(err).Msg("failed to delete interned message")
		return
	}
}

type MessageToSend struct {
	ChannelID string
	Req       CreateMessageRequest
	ExpiresAt time.Time
}

func SendMessages(
	ctx context.Context,
	conn db.ConnOrTx,
	msgs ...MessageToSend,
) error {
	tx, err := conn.Begin(ctx)
	if err != nil {
		return oops.New(err, "failed to start transaction")
	}
	defer tx.Rollback(ctx)

	for _, msg := range msgs {
		if msg.ExpiresAt.IsZero() {
			msg.ExpiresAt = time.Now().Add(30 * time.Second)
		}

		reqBytes, err := json.Marshal(msg.Req)
		if err != nil {
			return oops.New(err, "failed to marshal Discord message to JSON")
		}

		_, err = tx.Exec(ctx,
			`
			INSERT INTO discord_outgoing_message (channel_id, payload_json, expires_at)
			VALUES ($1, $2, $3)
			`,
			msg.ChannelID,
			string(reqBytes),
			msg.ExpiresAt,
		)
		if err != nil {
			return oops.New(err, "failed to save outgoing Discord message to the database")
		}
	}

	err = tx.Commit(ctx)
	if err != nil {
		return oops.New(err, "failed to commit outgoing Discord messages")
	}

	// Notify the sender that messages are ready to go
	select {
	case outgoingMessagesReady <- struct{}{}:
	default:
	}

	return nil
}

package discord

import (
	"encoding/json"
	"fmt"
	"time"
)

type Opcode int

// https://discord.com/developers/docs/topics/opcodes-and-status-codes#gateway-gateway-opcodes
// NOTE(ben): I'm not using iota because 5 is missing
const (
	OpcodeDispatch            Opcode = 0
	OpcodeHeartbeat           Opcode = 1
	OpcodeIdentify            Opcode = 2
	OpcodePresenceUpdate      Opcode = 3
	OpcodeVoiceStateUpdate    Opcode = 4
	OpcodeResume              Opcode = 6
	OpcodeReconnect           Opcode = 7
	OpcodeRequestGuildMembers Opcode = 8
	OpcodeInvalidSession      Opcode = 9
	OpcodeHello               Opcode = 10
	OpcodeHeartbeatACK        Opcode = 11
)

type Intent int

// https://discord.com/developers/docs/topics/gateway#list-of-intents
// NOTE(ben): I'm not using iota because the opcode thing made me paranoid
const (
	IntentGuilds                 Intent = 1 << 0
	IntentGuildMembers           Intent = 1 << 1
	IntentGuildBans              Intent = 1 << 2
	IntentGuildEmojisAndStickers Intent = 1 << 3
	IntentGuildIntegrations      Intent = 1 << 4
	IntentGuildWebhooks          Intent = 1 << 5
	IntentGuildInvites           Intent = 1 << 6
	IntentGuildVoiceStates       Intent = 1 << 7
	IntentGuildPresences         Intent = 1 << 8
	IntentGuildMessages          Intent = 1 << 9
	IntentGuildMessageReactions  Intent = 1 << 10
	IntentGuildMessageTyping     Intent = 1 << 11
	IntentDirectMessages         Intent = 1 << 12
	IntentDirectMessageReactions Intent = 1 << 13
	IntentDirectMessageTyping    Intent = 1 << 14
)

type GatewayMessage struct {
	Opcode         Opcode      `json:"op"`
	Data           interface{} `json:"d"`
	SequenceNumber *int        `json:"s,omitempty"`
	EventName      *string     `json:"t,omitempty"`
}

func (m *GatewayMessage) ToJSON() []byte {
	mBytes, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}

	return mBytes
}

type Hello struct {
	HeartbeatIntervalMs int `json:"heartbeat_interval"`
}

func HelloFromMap(m interface{}) Hello {
	return Hello{
		HeartbeatIntervalMs: int(m.(map[string]interface{})["heartbeat_interval"].(float64)),
	}
}

type Identify struct {
	Token      string                       `json:"token"`
	Properties IdentifyConnectionProperties `json:"properties"`
	Intents    Intent                       `json:"intents"`
}

type IdentifyConnectionProperties struct {
	OS      string `json:"$os"`
	Browser string `json:"$browser"`
	Device  string `json:"$device"`
}

type Ready struct {
	GatewayVersion int    `json:"v"`
	User           User   `json:"user"`
	SessionID      string `json:"session_id"`
}

func ReadyFromMap(m interface{}) Ready {
	mmap := m.(map[string]interface{})

	return Ready{
		GatewayVersion: int(mmap["v"].(float64)),
		User:           *UserFromMap(mmap["user"], ""),
		SessionID:      mmap["session_id"].(string),
	}
}

type Resume struct {
	Token          string `json:"token"`
	SessionID      string `json:"session_id"`
	SequenceNumber int    `json:"seq"`
}

// https://discord.com/developers/docs/topics/gateway#message-delete
type MessageDelete struct {
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	GuildID   string `json:"guild_id"`
}

func MessageDeleteFromMap(m interface{}) MessageDelete {
	mmap := m.(map[string]interface{})

	return MessageDelete{
		ID:        mmap["id"].(string),
		ChannelID: mmap["channel_id"].(string),
		GuildID:   maybeString(mmap, "guild_id"),
	}
}

// https://discord.com/developers/docs/topics/gateway#message-delete
type MessageBulkDelete struct {
	IDs       []string `json:"ids"`
	ChannelID string   `json:"channel_id"`
	GuildID   string   `json:"guild_id"`
}

func MessageBulkDeleteFromMap(m interface{}) MessageBulkDelete {
	mmap := m.(map[string]interface{})

	iids := mmap["ids"].([]interface{})
	ids := make([]string, len(iids))
	for i, iid := range iids {
		ids[i] = iid.(string)
	}

	return MessageBulkDelete{
		IDs:       ids,
		ChannelID: mmap["channel_id"].(string),
		GuildID:   maybeString(mmap, "guild_id"),
	}
}

type ChannelType int

// https://discord.com/developers/docs/resources/channel#channel-object-channel-types
const (
	ChannelTypeGuildText          ChannelType = 0
	ChannelTypeDM                 ChannelType = 1
	ChannelTypeGuildVoice         ChannelType = 2
	ChannelTypeGroupDM            ChannelType = 3
	ChannelTypeGuildCategory      ChannelType = 4
	ChannelTypeGuildNews          ChannelType = 5
	ChannelTypeGuildStore         ChannelType = 6
	ChannelTypeGuildNewsThread    ChannelType = 10
	ChannelTypeGuildPublicThread  ChannelType = 11
	ChannelTypeGuildPrivateThread ChannelType = 12
	ChannelTypeGuildStageVoice    ChannelType = 13
)

// https://discord.com/developers/docs/topics/permissions#role-object
type Role struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	// more fields not yet present
}

func RoleFromMap(m interface{}, k string) *Role {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	r := &Role{
		ID:   mmap["id"].(string),
		Name: mmap["name"].(string),
	}

	return r
}

// https://discord.com/developers/docs/resources/channel#channel-object
type Channel struct {
	ID      string      `json:"id"`
	Type    ChannelType `json:"type"`
	GuildID string      `json:"guild_id"`
	Name    string      `json:"name"`
	// More fields not yet present
}

func ChannelFromMap(m interface{}, k string) *Channel {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	c := &Channel{
		ID:      mmap["id"].(string),
		Type:    ChannelType(mmap["type"].(float64)),
		GuildID: maybeString(mmap, "guild_id"),
		Name:    maybeString(mmap, "name"),
	}

	return c
}

type MessageType int

// https://discord.com/developers/docs/resources/channel#message-object-message-types
const (
	MessageTypeDefault MessageType = 0

	MessageTypeRecipientAdd    MessageType = 1
	MessageTypeRecipientRemove MessageType = 2
	MessageTypeCall            MessageType = 3

	MessageTypeChannelNameChange    MessageType = 4
	MessageTypeChannelIconChange    MessageType = 5
	MessageTypeChannelPinnedMessage MessageType = 6

	MessageTypeGuildMemberJoin MessageType = 7

	MessageTypeUserPremiumGuildSubscription      MessageType = 8
	MessageTypeUserPremiumGuildSubscriptionTier1 MessageType = 9
	MessageTypeUserPremiumGuildSubscriptionTier2 MessageType = 10
	MessageTypeUserPremiumGuildSubscriptionTier3 MessageType = 11

	MessageTypeChannelFollowAdd MessageType = 12

	MessageTypeGuildDiscoveryDisqualified              MessageType = 14
	MessageTypeGuildDiscoveryRequalified               MessageType = 15
	MessageTypeGuildDiscoveryGracePeriodInitialWarning MessageType = 16
	MessageTypeGuildDiscoveryGracePeriodFinalWarning   MessageType = 17

	MessageTypeThreadCreated        MessageType = 18
	MessageTypeReply                MessageType = 19
	MessageTypeApplicationCommand   MessageType = 20
	MessageTypeThreadStarterMessage MessageType = 21
	MessageTypeGuildInviteReminder  MessageType = 22
)

// https://discord.com/developers/docs/resources/channel#message-object
type Message struct {
	ID        string      `json:"id"`
	ChannelID string      `json:"channel_id"`
	GuildID   *string     `json:"guild_id"`
	Content   string      `json:"content"`
	Author    *User       `json:"author"` // note that this may not be an actual valid user (see the docs)
	Timestamp string      `json:"timestamp"`
	Type      MessageType `json:"type"`

	Attachments []Attachment `json:"attachments"`
	Embeds      []Embed      `json:"embeds"`

	originalMap map[string]interface{}
}

func (m *Message) JumpURL() string {
	guildStr := "@me"
	if m.GuildID != nil {
		guildStr = *m.GuildID
	}
	return fmt.Sprintf("https://discord.com/channels/%s/%s/%s", guildStr, m.ChannelID, m.ID)
}

func (m *Message) Time() time.Time {
	t, err := time.Parse(time.RFC3339Nano, m.Timestamp)
	if err != nil {
		panic(err)
	}
	return t
}

func (m *Message) ShortString() string {
	return fmt.Sprintf("%s / %s: \"%s\" (%d attachments, %d embeds)", m.Timestamp, m.Author.Username, m.Content, len(m.Attachments), len(m.Embeds))
}

func (m *Message) OriginalHasFields(fields ...string) bool {
	if m.originalMap == nil {
		// If we don't know, we assume the fields are there.
		// Usually this is because it came from their API, where we
		// always have all fields.
		return true
	}

	for _, field := range fields {
		_, ok := m.originalMap[field]
		if !ok {
			return false
		}
	}
	return true
}

func MessageFromMap(m interface{}, k string) *Message {
	/*
		Some gateway events, like MESSAGE_UPDATE, do not contain the
		entire message body. So we need to be defensive on all fields here,
		except the most basic identifying information.
	*/

	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	msg := Message{
		ID:        mmap["id"].(string),
		ChannelID: mmap["channel_id"].(string),
		GuildID:   maybeStringP(mmap, "guild_id"),
		Content:   maybeString(mmap, "content"),
		Author:    UserFromMap(m, "author"),
		Timestamp: maybeString(mmap, "timestamp"),
		Type:      MessageType(maybeInt(mmap, "type")),

		originalMap: mmap,
	}

	if iattachments, ok := mmap["attachments"]; ok {
		attachments := iattachments.([]interface{})
		for _, iattachment := range attachments {
			msg.Attachments = append(msg.Attachments, *AttachmentFromMap(iattachment, ""))
		}
	}

	if iembeds, ok := mmap["embeds"]; ok {
		embeds := iembeds.([]interface{})
		for _, iembed := range embeds {
			msg.Embeds = append(msg.Embeds, *EmbedFromMap(iembed))
		}
	}

	return &msg
}

// https://discord.com/developers/docs/resources/user#user-object
type User struct {
	ID            string  `json:"id"`
	Username      string  `json:"username"`
	Discriminator string  `json:"discriminator"`
	Avatar        *string `json:"avatar"`
	IsBot         bool    `json:"bot"`
	Locale        string  `json:"locale"`
}

func UserFromMap(m interface{}, k string) *User {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	u := User{
		ID:            mmap["id"].(string),
		Username:      mmap["username"].(string),
		Discriminator: mmap["discriminator"].(string),
	}

	if isBot, ok := mmap["bot"]; ok {
		u.IsBot = isBot.(bool)
	}

	return &u
}

type Guild struct {
	ID string `json:"id"`
	// Who cares about the rest tbh
}

func GuildFromMap(m interface{}, k string) *Guild {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	g := Guild{
		ID: mmap["id"].(string),
	}

	return &g
}

// https://discord.com/developers/docs/resources/guild#guild-member-object
type GuildMember struct {
	User *User   `json:"user"`
	Nick *string `json:"nick"`
	// more fields not yet handled here
}

func (gm *GuildMember) DisplayName() string {
	if gm.Nick != nil {
		return *gm.Nick
	}
	if gm.User != nil {
		return gm.User.Username
	}
	return "<UNKNOWN USER>"
}

func GuildMemberFromMap(m interface{}, k string) *GuildMember {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	gm := &GuildMember{
		User: UserFromMap(m, "user"),
		Nick: maybeStringP(mmap, "nick"),
	}

	return gm
}

// https://discord.com/developers/docs/resources/channel#attachment-object
type Attachment struct {
	ID          string  `json:"id"`
	Filename    string  `json:"filename"`
	ContentType *string `json:"content_type"`
	Size        int     `json:"size"`
	Url         string  `json:"url"`
	ProxyUrl    string  `json:"proxy_url"`
	Height      *int    `json:"height"`
	Width       *int    `json:"width"`
}

func AttachmentFromMap(m interface{}, k string) *Attachment {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	a := Attachment{
		ID:          mmap["id"].(string),
		Filename:    mmap["filename"].(string),
		ContentType: maybeStringP(mmap, "content_type"),
		Size:        int(mmap["size"].(float64)),
		Url:         mmap["url"].(string),
		ProxyUrl:    mmap["proxy_url"].(string),
		Height:      maybeIntP(mmap, "height"),
		Width:       maybeIntP(mmap, "width"),
	}

	return &a
}

// https://discord.com/developers/docs/resources/channel#embed-object
type Embed struct {
	Title       *string         `json:"title"`
	Type        *string         `json:"type"`
	Description *string         `json:"description"`
	Url         *string         `json:"url"`
	Timestamp   *string         `json:"timestamp"`
	Color       *int            `json:"color"`
	Footer      *EmbedFooter    `json:"footer"`
	Image       *EmbedImage     `json:"image"`
	Thumbnail   *EmbedThumbnail `json:"thumbnail"`
	Video       *EmbedVideo     `json:"video"`
	Provider    *EmbedProvider  `json:"provider"`
	Author      *EmbedAuthor    `json:"author"`
	Fields      []EmbedField    `json:"fields"`
}

type EmbedFooter struct {
	Text         string  `json:"text"`
	IconUrl      *string `json:"icon_url"`
	ProxyIconUrl *string `json:"proxy_icon_url"`
}

type EmbedImageish struct {
	Url      *string `json:"url"`
	ProxyUrl *string `json:"proxy_url"`
	Height   *int    `json:"height"`
	Width    *int    `json:"width"`
}

type EmbedImage struct {
	EmbedImageish
}

type EmbedThumbnail struct {
	EmbedImageish
}

type EmbedVideo struct {
	EmbedImageish
}

type EmbedProvider struct {
	Name *string `json:"name"`
	Url  *string `json:"url"`
}

type EmbedAuthor struct {
	Name         *string `json:"name"`
	Url          *string `json:"url"`
	IconUrl      *string `json:"icon_url"`
	ProxyIconUrl *string `json:"proxy_icon_url"`
}

type EmbedField struct {
	Name   string `json:"name"`
	Value  string `json:"value"`
	Inline *bool  `json:"inline"`
}

func EmbedFromMap(m interface{}) *Embed {
	mmap := m.(map[string]interface{})
	if mmap == nil {
		return nil
	}

	e := Embed{
		Title:       maybeStringP(mmap, "title"),
		Type:        maybeStringP(mmap, "type"),
		Description: maybeStringP(mmap, "description"),
		Url:         maybeStringP(mmap, "url"),
		Timestamp:   maybeStringP(mmap, "timestamp"),
		Color:       maybeIntP(mmap, "color"),
		Footer:      EmbedFooterFromMap(mmap, "footer"),
		Image:       EmbedImageFromMap(mmap, "image"),
		Thumbnail:   EmbedThumbnailFromMap(mmap, "thumbnail"),
		Video:       EmbedVideoFromMap(mmap, "video"),
		Provider:    EmbedProviderFromMap(mmap, "provider"),
		Author:      EmbedAuthorFromMap(mmap, "author"),
		Fields:      EmbedFieldsFromMap(mmap, "fields"),
	}

	return &e
}

func EmbedFooterFromMap(m map[string]interface{}, k string) *EmbedFooter {
	f, ok := m[k]
	if !ok {
		return nil
	}
	fMap, ok := f.(map[string]interface{})
	if !ok {
		return nil
	}

	return &EmbedFooter{
		Text:         maybeString(fMap, "text"),
		IconUrl:      maybeStringP(fMap, "icon_url"),
		ProxyIconUrl: maybeStringP(fMap, "proxy_icon_url"),
	}
}

func EmbedImageFromMap(m map[string]interface{}, k string) *EmbedImage {
	val, ok := m[k]
	if !ok {
		return nil
	}
	valMap, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}

	return &EmbedImage{
		EmbedImageish: EmbedImageish{
			Url:      maybeStringP(valMap, "url"),
			ProxyUrl: maybeStringP(valMap, "proxy_url"),
			Height:   maybeIntP(valMap, "height"),
			Width:    maybeIntP(valMap, "width"),
		},
	}
}

func EmbedThumbnailFromMap(m map[string]interface{}, k string) *EmbedThumbnail {
	val, ok := m[k]
	if !ok {
		return nil
	}
	valMap, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}

	return &EmbedThumbnail{
		EmbedImageish: EmbedImageish{
			Url:      maybeStringP(valMap, "url"),
			ProxyUrl: maybeStringP(valMap, "proxy_url"),
			Height:   maybeIntP(valMap, "height"),
			Width:    maybeIntP(valMap, "width"),
		},
	}
}

func EmbedVideoFromMap(m map[string]interface{}, k string) *EmbedVideo {
	val, ok := m[k]
	if !ok {
		return nil
	}
	valMap, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}

	return &EmbedVideo{
		EmbedImageish: EmbedImageish{
			Url:      maybeStringP(valMap, "url"),
			ProxyUrl: maybeStringP(valMap, "proxy_url"),
			Height:   maybeIntP(valMap, "height"),
			Width:    maybeIntP(valMap, "width"),
		},
	}
}

func EmbedProviderFromMap(m map[string]interface{}, k string) *EmbedProvider {
	val, ok := m[k]
	if !ok {
		return nil
	}
	valMap, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}

	return &EmbedProvider{
		Name: maybeStringP(valMap, "name"),
		Url:  maybeStringP(valMap, "url"),
	}
}

func EmbedAuthorFromMap(m map[string]interface{}, k string) *EmbedAuthor {
	val, ok := m[k]
	if !ok {
		return nil
	}
	valMap, ok := val.(map[string]interface{})
	if !ok {
		return nil
	}

	return &EmbedAuthor{
		Name: maybeStringP(valMap, "name"),
		Url:  maybeStringP(valMap, "url"),
	}
}

func EmbedFieldsFromMap(m map[string]interface{}, k string) []EmbedField {
	val, ok := m[k]
	if !ok {
		return nil
	}
	valSlice, ok := val.([]interface{})
	if !ok {
		return nil
	}

	var result []EmbedField
	for _, innerVal := range valSlice {
		valMap, ok := innerVal.(map[string]interface{})
		if !ok {
			continue
		}

		result = append(result, EmbedField{
			Name:   maybeString(valMap, "name"),
			Value:  maybeString(valMap, "value"),
			Inline: maybeBoolP(valMap, "inline"),
		})
	}

	return result
}

// Data is always present on application command and message component interaction types. It is optional for future-proofing against new interaction types.
//
// Member is sent when the interaction is invoked in a guild, and User is sent when invoked in a DM.
//
// See https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-structure
type Interaction struct {
	ID            string           `json:"id"`             // id of the interaction
	ApplicationID string           `json:"application_id"` // id of the application this interaction is for
	Type          InteractionType  `json:"type"`           // the type of interaction
	Data          *InteractionData `json:"data"`           // the command data payload
	GuildID       string           `json:"guild_id"`       // the guild it was sent from
	ChannelID     string           `json:"channel_id"`     // the channel it was sent from
	Member        *GuildMember     `json:"member"`         // guild member data for the invoking user, including permissions
	User          *User            `json:"user"`           // user object for the invoking user, if invoked in a DM
	Token         string           `json:"token"`          // a continuation token for responding to the interaction
	Version       int              `json:"version"`        // read-only property, always 1
	Message       *Message         `json:"message"`        // for components, the message they were attached to
}

// https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-type
type InteractionType int

const (
	InteractionTypePing               InteractionType = 1
	InteractionTypeApplicationCommand InteractionType = 2
	InteractionTypeMessageComponent   InteractionType = 3
)

// See https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-interaction-data-structure
type InteractionData struct {
	// Fields for Application Commands
	// See https://discord.com/developers/docs/interactions/application-commands#application-command-object-application-command-structure

	ID       string                                    `json:"id"`       // the ID of the invoked command
	Name     string                                    `json:"name"`     // the name of the invoked command
	Type     ApplicationCommandType                    `json:"type"`     // the type of the invoked command
	Resolved *ResolvedData                             `json:"resolved"` // converted users + roles + channels
	Options  []ApplicationCommandInteractionDataOption `json:"options"`  // the params + values from the user

	// Fields for Components
	// TODO

	// Fields for User Command and Message Command
	TargetID string `json:"target_id"` // id the of user or message targetted by a user or message command
}

// See https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-object-resolved-data-structure
type ResolvedData struct {
	Users    map[string]User        `json:"users"`
	Members  map[string]GuildMember `json:"members"` // Partial Member objects are missing user, deaf and mute fields. If data for a Member is included, data for its corresponding User will also be included.
	Roles    map[string]Role        `json:"roles"`
	Channels map[string]Channel     `json:"channels"` // Partial Channel objects only have id, name, type and permissions fields. Threads will also have thread_metadata and parent_id fields.
	Messages map[string]Message     `json:"messages"`
}

// See https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-interaction-response-structure
type InteractionResponse struct {
	Type InteractionCallbackType  `json:"type"` // the type of response
	Data *InteractionCallbackData `json:"data"` // an optional response message
}

// See https://discord.com/developers/docs/interactions/receiving-and-responding#interaction-response-object-interaction-callback-type
type InteractionCallbackType int

const (
	InteractionCallbackTypePong                             InteractionCallbackType = 1 // ACK a Ping
	InteractionCallbackTypeChannelMessageWithSource         InteractionCallbackType = 4 // respond to an interaction with a message
	InteractionCallbackTypeDeferredChannelMessageWithSource InteractionCallbackType = 5 // ACK an interaction and edit a response later, the user sees a loading state
	InteractionCallbackTypeDeferredUpdateMessage            InteractionCallbackType = 6 // for components, ACK an interaction and edit the original message later; the user does not see a loading state
	InteractionCallbackTypeUpdateMessage                    InteractionCallbackType = 7 // for components, edit the message the component was attached to
)

type InteractionCallbackData struct {
	TTS     bool    `json:"bool,omitempty"`
	Content string  `json:"content,omitempty"`
	Embeds  []Embed `json:"embeds,omitempty"`
	// TODO: Allowed mentions
	Flags InteractionCallbackDataFlags `json:"flags,omitempty"`
	// TODO: Components
}

type InteractionCallbackDataFlags int

const (
	FlagEphemeral InteractionCallbackDataFlags = 1 << 6
)

type ApplicationCommandType int

const (
	ApplicationCommandTypeChatInput ApplicationCommandType       = 1 // Slash commands; a text-based command that shows up when a user types `/`
	ApplicationCommandTypeUser      ApplicationCommandType       = 2 // A UI-based command that shows up when you right click or tap on a user
	ApplicationCommandTypeMessage   ApplicationCommandOptionType = 3 // A UI-based command that shows up when you right click or tap on a message
)

// Required `options` must be listed before optional options
type ApplicationCommandOption struct {
	Type        ApplicationCommandOptionType     `json:"type"`        // the type of option
	Name        string                           `json:"name"`        // 1-32 character name
	Description string                           `json:"description"` // 1-100 character description
	Required    bool                             `json:"required"`    // if the parameter is required or optional--default false
	Choices     []ApplicationCommandOptionChoice `json:"choices"`     // choices for STRING, INTEGER, and NUMBER types for the user to pick from, max 25
	Options     []ApplicationCommandOption       `json:"options"`     // if the option is a subcommand or subcommand group type, this nested options will be the parameters
}

type ApplicationCommandOptionType int

const (
	ApplicationCommandOptionTypeSubCommand      ApplicationCommandOptionType = 1
	ApplicationCommandOptionTypeSubCommandGroup ApplicationCommandOptionType = 2
	ApplicationCommandOptionTypeString          ApplicationCommandOptionType = 3
	ApplicationCommandOptionTypeInteger         ApplicationCommandOptionType = 4 // Any integer between -2^53 and 2^53
	ApplicationCommandOptionTypeBoolean         ApplicationCommandOptionType = 5
	ApplicationCommandOptionTypeUser            ApplicationCommandOptionType = 6
	ApplicationCommandOptionTypeChannel         ApplicationCommandOptionType = 7 // Includes all channel types + categories
	ApplicationCommandOptionTypeRole            ApplicationCommandOptionType = 8
	ApplicationCommandOptionTypeMentionable     ApplicationCommandOptionType = 9  // Includes users and roles
	ApplicationCommandOptionTypeNumber          ApplicationCommandOptionType = 10 // Any double between -2^53 and 2^53
)

// If you specify `choices` for an option, they are the only valid values for a user to pick
type ApplicationCommandOptionChoice struct {
	Name  string      `json:"name"`  // 1-100 character choice name
	Value interface{} `json:"value"` // value of the choice, up to 100 characters if string
}

// All options have names, and an option can either be a parameter and input
// value--in which case Value will be set--or it can denote a subcommand or
// group--in which case it will contain a top-level key and another array
// of Options.
//
// Value and Options are mutually exclusive.
type ApplicationCommandInteractionDataOption struct {
	Name    string                                    `json:"name"`
	Type    ApplicationCommandOptionType              `json:"type"`
	Value   interface{}                               `json:"value"`   // the value of the pair
	Options []ApplicationCommandInteractionDataOption `json:"options"` // present if this option is a group or subcommand
}

func InteractionFromMap(m interface{}, k string) *Interaction {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	i := &Interaction{
		ID:            mmap["id"].(string),
		ApplicationID: mmap["application_id"].(string),
		Type:          InteractionType(mmap["type"].(float64)),
		Data:          InteractionDataFromMap(m, "data"),
		GuildID:       maybeString(mmap, "guild_id"),
		ChannelID:     maybeString(mmap, "channel_id"),
		Member:        GuildMemberFromMap(mmap, "member"),
		User:          UserFromMap(mmap, "user"),
		Token:         mmap["token"].(string),
		Version:       int(mmap["version"].(float64)),
		Message:       MessageFromMap(mmap, "message"),
	}

	return i
}

func InteractionDataFromMap(m interface{}, k string) *InteractionData {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	d := &InteractionData{
		ID:       mmap["id"].(string),
		Name:     mmap["name"].(string),
		Type:     ApplicationCommandType(mmap["type"].(float64)),
		Resolved: ResolvedDataFromMap(mmap, "resolved"),
		TargetID: maybeString(mmap, "target_id"),
	}

	if ioptions, ok := mmap["options"]; ok {
		options := ioptions.([]interface{})
		for _, ioption := range options {
			d.Options = append(d.Options, *ApplicationCommandInteractionDataOptionFromMap(ioption, ""))
		}
	}

	return d
}

func ResolvedDataFromMap(m interface{}, k string) *ResolvedData {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	d := &ResolvedData{}

	if iusers, ok := mmap["users"]; ok {
		users := iusers.(map[string]interface{})
		d.Users = make(map[string]User)
		for id, iuser := range users {
			d.Users[id] = *UserFromMap(iuser, "")
		}
	}

	if imembers, ok := mmap["members"]; ok {
		members := imembers.(map[string]interface{})
		d.Members = make(map[string]GuildMember)
		for id, imember := range members {
			member := *GuildMemberFromMap(imember, "")
			user := d.Users[id]
			member.User = &user
			d.Members[id] = member
		}
	}

	if iroles, ok := mmap["roles"]; ok {
		roles := iroles.(map[string]interface{})
		d.Roles = make(map[string]Role)
		for id, irole := range roles {
			d.Roles[id] = *RoleFromMap(irole, "")
		}
	}

	if ichannels, ok := mmap["channels"]; ok {
		channels := ichannels.(map[string]interface{})
		d.Channels = make(map[string]Channel)
		for id, ichannel := range channels {
			d.Channels[id] = *ChannelFromMap(ichannel, "")
		}
	}

	if imessages, ok := mmap["messages"]; ok {
		messages := imessages.(map[string]interface{})
		d.Messages = make(map[string]Message)
		for id, imessage := range messages {
			d.Messages[id] = *MessageFromMap(imessage, "")
		}
	}

	return d
}

func ApplicationCommandInteractionDataOptionFromMap(m interface{}, k string) *ApplicationCommandInteractionDataOption {
	mmap := maybeGetKey(m, k)
	if mmap == nil {
		return nil
	}

	o := &ApplicationCommandInteractionDataOption{
		Name:  mmap["name"].(string),
		Type:  ApplicationCommandOptionType(mmap["type"].(float64)),
		Value: mmap["value"],
	}

	if ioptions, ok := mmap["options"]; ok {
		options := ioptions.([]interface{})
		for _, ioption := range options {
			o.Options = append(o.Options, *ApplicationCommandInteractionDataOptionFromMap(ioption, ""))
		}
	}

	return o
}

// If called without a key, returns m. Otherwise, returns m[k].
// If m[k] does not exist, returns nil.
//
// The intent is to allow the ThingFromMap functions to be flexibly called,
// either with the data in question as the root (no key) or as a child of
// another object (with a key).
func maybeGetKey(m interface{}, k string) map[string]interface{} {
	if k == "" {
		return m.(map[string]interface{})
	} else {
		mmap := m.(map[string]interface{})
		if mk, ok := mmap[k]; ok {
			return mk.(map[string]interface{})
		} else {
			return nil
		}
	}
}

func maybeString(m map[string]interface{}, k string) string {
	val, ok := m[k]
	if !ok {
		return ""
	}
	return val.(string)
}

func maybeStringP(m map[string]interface{}, k string) *string {
	val, ok := m[k]
	if !ok || val == nil {
		return nil
	}
	strval := val.(string)
	return &strval
}

func maybeInt(m map[string]interface{}, k string) int {
	val, ok := m[k]
	if !ok {
		return 0
	}
	return int(val.(float64))
}

func maybeIntP(m map[string]interface{}, k string) *int {
	val, ok := m[k]
	if !ok || val == nil {
		return nil
	}
	intval := int(val.(float64))
	return &intval
}

func maybeBool(m map[string]interface{}, k string) bool {
	val, ok := m[k]
	if !ok {
		return false
	}
	return val.(bool)
}

func maybeBoolP(m map[string]interface{}, k string) *bool {
	val, ok := m[k]
	if !ok || val == nil {
		return nil
	}
	boolval := val.(bool)
	return &boolval
}

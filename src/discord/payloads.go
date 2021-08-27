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

	// TODO: check if the payload is too big, either here or where we actually send
	// https://discord.com/developers/docs/topics/gateway#sending-payloads

	return mBytes
}

type Hello struct {
	HeartbeatIntervalMs int `json:"heartbeat_interval"`
}

func HelloFromMap(m interface{}) Hello {
	// TODO: This should probably have some error handling, right?
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
		User:           UserFromMap(mmap["user"]),
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

// https://discord.com/developers/docs/resources/channel#channel-object
type Channel struct {
	ID          string      `json:"id"`
	Type        ChannelType `json:"type"`
	GuildID     string      `json:"guild_id"`
	Name        string      `json:"name"`
	Receipients []User      `json:"recipients"`
	OwnerID     User        `json:"owner_id"`
	ParentID    *string     `json:"parent_id"`
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
	Author    User        `json:"author"` // note that this may not be an actual valid user (see the docs)
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

func MessageFromMap(m interface{}) Message {
	/*
		Some gateway events, like MESSAGE_UPDATE, do not contain the
		entire message body. So we need to be defensive on all fields here,
		except the most basic identifying information.
	*/

	mmap := m.(map[string]interface{})
	msg := Message{
		ID:        mmap["id"].(string),
		ChannelID: mmap["channel_id"].(string),
		GuildID:   maybeStringP(mmap, "guild_id"),
		Content:   maybeString(mmap, "content"),
		Timestamp: maybeString(mmap, "timestamp"),
		Type:      MessageType(maybeInt(mmap, "type")),

		originalMap: mmap,
	}

	if author, ok := mmap["author"]; ok {
		msg.Author = UserFromMap(author)
	}

	if iattachments, ok := mmap["attachments"]; ok {
		attachments := iattachments.([]interface{})
		for _, iattachment := range attachments {
			msg.Attachments = append(msg.Attachments, AttachmentFromMap(iattachment))
		}
	}

	if iembeds, ok := mmap["embeds"]; ok {
		embeds := iembeds.([]interface{})
		for _, iembed := range embeds {
			msg.Embeds = append(msg.Embeds, EmbedFromMap(iembed))
		}
	}

	return msg
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

func UserFromMap(m interface{}) User {
	mmap := m.(map[string]interface{})

	u := User{
		ID:            mmap["id"].(string),
		Username:      mmap["username"].(string),
		Discriminator: mmap["discriminator"].(string),
	}

	if isBot, ok := mmap["bot"]; ok {
		u.IsBot = isBot.(bool)
	}

	return u
}

// https://discord.com/developers/docs/resources/guild#guild-member-object
type GuildMember struct {
	User *User   `json:"user"`
	Nick *string `json:"nick"`
	// more fields not yet handled here
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

func AttachmentFromMap(m interface{}) Attachment {
	mmap := m.(map[string]interface{})
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

	return a
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

func EmbedFromMap(m interface{}) Embed {
	mmap := m.(map[string]interface{})

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

	return e
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

func maybeString(m map[string]interface{}, k string) string {
	val, ok := m[k]
	if !ok {
		return ""
	}
	return val.(string)
}

func maybeStringP(m map[string]interface{}, k string) *string {
	val, ok := m[k]
	if !ok {
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
	if !ok {
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
	if !ok {
		return nil
	}
	boolval := val.(bool)
	return &boolval
}

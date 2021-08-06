package discord

import (
	"encoding/json"
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

type ChannelType int

const (
	ChannelTypeGuildext           ChannelType = 0
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
	ID        string `json:"id"`
	ChannelID string `json:"channel_id"`
	Content   string `json:"content"`
	Author    *User  `json:"author"` // note that this may not be an actual valid user (see the docs)
	// TODO: Author info
	// TODO: Timestamp parsing, yay
	Type MessageType `json:"type"`

	Attachments []Attachment `json:"attachments"`

	originalMap map[string]interface{}
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
		Content:   maybeString(mmap, "content"),
		Type:      MessageType(maybeInt(mmap, "type")),

		originalMap: mmap,
	}

	if author, ok := mmap["author"]; ok {
		u := UserFromMap(author)
		msg.Author = &u
	}

	if iattachments, ok := mmap["attachments"]; ok {
		attachments := iattachments.([]interface{})
		for _, iattachment := range attachments {
			msg.Attachments = append(msg.Attachments, AttachmentFromMap(iattachment))
		}
	}

	return msg
}

// https://discord.com/developers/docs/resources/user#user-object
type User struct {
	ID            string `json:"id"`
	Username      string `json:"username"`
	Discriminator string `json:"discriminator"`
	IsBot         bool   `json:"bot"`
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

type Attachment struct {
	ID          string `json:"id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	Size        int    `json:"size"`
	Url         string `json:"url"`
	ProxyUrl    string `json:"proxy_url"`
	Height      *int   `json:"height"`
	Width       *int   `json:"width"`
}

func AttachmentFromMap(m interface{}) Attachment {
	mmap := m.(map[string]interface{})
	a := Attachment{
		ID:          mmap["id"].(string),
		Filename:    mmap["filename"].(string),
		ContentType: maybeString(mmap, "content_type"),
		Size:        int(mmap["size"].(float64)),
		Url:         mmap["url"].(string),
		ProxyUrl:    mmap["proxy_url"].(string),
		Height:      maybeIntP(mmap, "height"),
		Width:       maybeIntP(mmap, "width"),
	}

	return a
}

func maybeString(m map[string]interface{}, k string) string {
	val, ok := m[k]
	if !ok {
		return ""
	}
	return val.(string)
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

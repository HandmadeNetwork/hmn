package discord

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMessageFromMap(t *testing.T) {
	t.Run("create", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testMessageCreate), &m))

		message := MessageFromMap(m, "")
		assert.Equal(t, "swag", message.Content)
		assert.Equal(t, "bvisness", message.Author.Username)
	})
	t.Run("update", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testMessageUpdate), &m))

		message := MessageFromMap(m, "")
		assert.Equal(t, "swank", message.Content)
		assert.Equal(t, "bvisness", message.Author.Username)
	})
	t.Run("delete", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testMessageDelete), &m))

		MessageDeleteFromMap(m)
	})
	t.Run("create with attachment", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testMessageCreate_Attachment), &m))

		message := MessageFromMap(m, "")
		assert.Len(t, message.Attachments, 1)
		assert.Equal(t, "legobrick.png", message.Attachments[0].Filename)
	})
	t.Run("delete attachment", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testMessageUpdate_Attachment), &m))

		message := MessageFromMap(m, "")
		assert.Len(t, message.Attachments, 0)
	})
	t.Run("create with embed", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testMessageCreate_Embed), &m))

		message := MessageFromMap(m, "")
		assert.Len(t, message.Embeds, 0) // no embeds in original message
	})
	t.Run("update with new embeds", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testMessageCreate_AddEmbed), &m))

		message := MessageFromMap(m, "")
		assert.Len(t, message.Embeds, 1)
		assert.Equal(t, "https://handmade.network/jam", *message.Embeds[0].Url)
	})
	t.Run("delete embeds", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testMessageCreate_DeleteEmbed), &m))

		message := MessageFromMap(m, "")
		assert.Len(t, message.Embeds, 0)
	})
}

func TestInteractionFromMap(t *testing.T) {
	t.Run("slash command", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testInteractionCreate_Slash), &m))

		i := InteractionFromMap(m, "")
		assert.Equal(t, "745036422834028565", i.ApplicationID)
		assert.Equal(t, "profile", i.Data.Name)
		assert.Equal(t, "132715550571888640", i.Data.Options[0].Value)

		userID := i.Data.Options[0].Value.(string)

		assert.Equal(t, "132715550571888640", i.Data.Resolved.Users[userID].ID)
		assert.Equal(t, "132715550571888640", i.Data.Resolved.Members[userID].User.ID)
	})
	t.Run("user command", func(t *testing.T) {
		var m interface{}
		assert.Nil(t, json.Unmarshal([]byte(testInteractionCreate_User), &m))

		i := InteractionFromMap(m, "")
		assert.Equal(t, "745036422834028565", i.ApplicationID)
		assert.Equal(t, "HMN Profile", i.Data.Name)
		assert.Equal(t, "132715550571888640", i.Data.TargetID)
		assert.Equal(t, "132715550571888640", i.Data.Resolved.Users[i.Data.TargetID].ID)
		assert.Equal(t, "132715550571888640", i.Data.Resolved.Members[i.Data.TargetID].User.ID)
	})
}

const testMessageCreate = `{
	"attachments": [],
	"author": {
		"avatar": "1963eacbf364164efce1c597dc66aeab",
		"discriminator": "3719",
		"id": "132715550571888640",
		"public_flags": 0,
		"username": "bvisness"
	},
	"channel_id": "605598627141910556",
	"components": [],
	"content": "swag",
	"edited_timestamp": null,
	"embeds": [],
	"flags": 0,
	"guild_id": "164936220651028480",
	"id": "891865665160507442",
	"member": {
		"avatar": null,
		"deaf": false,
		"hoisted_role": null,
		"is_pending": false,
		"joined_at": "2016-03-31T03:17:39.375000+00:00",
		"mute": false,
		"nick": null,
		"pending": false,
		"premium_since": null,
		"roles": [
			"876685379770646538"
		]
	},
	"mention_everyone": false,
	"mention_roles": [],
	"mentions": [],
	"nonce": "891865663746867200",
	"pinned": false,
	"referenced_message": null,
	"timestamp": "2021-09-27T01:55:44.637000+00:00",
	"tts": false,
	"type": 0
}`

const testMessageUpdate = `{
	"attachments": [],
	"author": {
		"avatar": "1963eacbf364164efce1c597dc66aeab",
		"discriminator": "3719",
		"id": "132715550571888640",
		"public_flags": 0,
		"username": "bvisness"
	},
	"channel_id": "605598627141910556",
	"components": [],
	"content": "swank",
	"edited_timestamp": "2021-09-27T01:54:00.260824+00:00",
	"embeds": [],
	"flags": 0,
	"guild_id": "164936220651028480",
	"id": "891865088183644170",
	"member": {
		"avatar": null,
		"deaf": false,
		"hoisted_role": null,
		"is_pending": false,
		"joined_at": "2016-03-31T03:17:39.375000+00:00",
		"mute": false,
		"nick": null,
		"pending": false,
		"premium_since": null,
		"roles": [
			"876685379770646538"
		]
	},
	"mention_everyone": false,
	"mention_roles": [],
	"mentions": [],
	"pinned": false,
	"timestamp": "2021-09-27T01:53:27.075000+00:00",
	"tts": false,
	"type": 0
}`

const testMessageDelete = `{
	"channel_id": "605598627141910556",
	"guild_id": "164936220651028480",
	"id": "891866242783248405"
}`

const testMessageCreate_Attachment = `{
	"attachments": [
		{
			"content_type": "image/png",
			"filename": "legobrick.png",
			"height": 385,
			"id": "891866614931288075",
			"proxy_url": "https://media.discordapp.net/attachments/605598627141910556/891866614931288075/legobrick.png",
			"size": 84148,
			"url": "https://cdn.discordapp.com/attachments/605598627141910556/891866614931288075/legobrick.png",
			"width": 512
		}
	],
	"author": {
		"avatar": "1963eacbf364164efce1c597dc66aeab",
		"discriminator": "3719",
		"id": "132715550571888640",
		"public_flags": 0,
		"username": "bvisness"
	},
	"channel_id": "605598627141910556",
	"components": [],
	"content": "our favorite",
	"edited_timestamp": null,
	"embeds": [],
	"flags": 0,
	"guild_id": "164936220651028480",
	"id": "891866615161950328",
	"member": {
		"avatar": null,
		"deaf": false,
		"hoisted_role": null,
		"is_pending": false,
		"joined_at": "2016-03-31T03:17:39.375000+00:00",
		"mute": false,
		"nick": null,
		"pending": false,
		"premium_since": null,
		"roles": [
			"876685379770646538"
		]
	},
	"mention_everyone": false,
	"mention_roles": [],
	"mentions": [],
	"pinned": false,
	"referenced_message": null,
	"timestamp": "2021-09-27T01:59:31.135000+00:00",
	"tts": false,
	"type": 0
}`

const testMessageUpdate_Attachment = `{
	"attachments": [],
	"author": {
		"avatar": "1963eacbf364164efce1c597dc66aeab",
		"discriminator": "3719",
		"id": "132715550571888640",
		"public_flags": 0,
		"username": "bvisness"
	},
	"channel_id": "605598627141910556",
	"components": [],
	"content": "our favorite",
	"edited_timestamp": "2021-09-27T01:59:37.262353+00:00",
	"embeds": [],
	"flags": 0,
	"guild_id": "164936220651028480",
	"id": "891866615161950328",
	"member": {
		"avatar": null,
		"deaf": false,
		"hoisted_role": null,
		"is_pending": false,
		"joined_at": "2016-03-31T03:17:39.375000+00:00",
		"mute": false,
		"nick": null,
		"pending": false,
		"premium_since": null,
		"roles": [
			"876685379770646538"
		]
	},
	"mention_everyone": false,
	"mention_roles": [],
	"mentions": [],
	"pinned": false,
	"timestamp": "2021-09-27T01:59:31.135000+00:00",
	"tts": false,
	"type": 0
}`

const testMessageCreate_Embed = `{
	"attachments": [],
	"author": {
		"avatar": "1963eacbf364164efce1c597dc66aeab",
		"discriminator": "3719",
		"id": "132715550571888640",
		"public_flags": 0,
		"username": "bvisness"
	},
	"channel_id": "605598627141910556",
	"components": [],
	"content": "https://handmade.network/jam",
	"edited_timestamp": null,
	"embeds": [],
	"flags": 0,
	"guild_id": "164936220651028480",
	"id": "891887149723566092",
	"member": {
		"avatar": null,
		"deaf": false,
		"hoisted_role": null,
		"is_pending": false,
		"joined_at": "2016-03-31T03:17:39.375000+00:00",
		"mute": false,
		"nick": null,
		"pending": false,
		"premium_since": null,
		"roles": [
			"876685379770646538"
		]
	},
	"mention_everyone": false,
	"mention_roles": [],
	"mentions": [],
	"nonce": "891887148477710336",
	"pinned": false,
	"referenced_message": null,
	"timestamp": "2021-09-27T03:21:06.956000+00:00",
	"tts": false,
	"type": 0
}`

const testMessageCreate_AddEmbed = `{
	"channel_id": "605598627141910556",
	"embeds": [
		{
			"description": "A one-week jam to bring a fresh perspective to old ideas. September 27 - October 3 on Handmade Network.",
			"provider": {
				"name": "Handmade.Network"
			},
			"thumbnail": {
				"height": 630,
				"proxy_url": "https://images-ext-2.discordapp.net/external/kqIj9PXmxlWUFDygt_2yMf5xKWEo9avdWs0gQcN0l2s/%3Fv%3D1632704138/https/handmade.network/public/wheeljam/opengraph.png",
				"url": "https://handmade.network/public/wheeljam/opengraph.png?v=1632704138",
				"width": 632
			},
			"type": "link",
			"url": "https://handmade.network/jam"
		}
	],
	"guild_id": "164936220651028480",
	"id": "891887149723566092"
}`

const testMessageCreate_DeleteEmbed = `{
	"attachments": [],
	"author": {
		"avatar": "1963eacbf364164efce1c597dc66aeab",
		"discriminator": "3719",
		"id": "132715550571888640",
		"public_flags": 0,
		"username": "bvisness"
	},
	"channel_id": "605598627141910556",
	"components": [],
	"content": "https://handmade.network/jam",
	"edited_timestamp": null,
	"embeds": [],
	"flags": 4,
	"guild_id": "164936220651028480",
	"id": "891887149723566092",
	"member": {
		"avatar": null,
		"deaf": false,
		"hoisted_role": null,
		"is_pending": false,
		"joined_at": "2016-03-31T03:17:39.375000+00:00",
		"mute": false,
		"nick": null,
		"pending": false,
		"premium_since": null,
		"roles": [
			"876685379770646538"
		]
	},
	"mention_everyone": false,
	"mention_roles": [],
	"mentions": [],
	"pinned": false,
	"timestamp": "2021-09-27T03:21:06.956000+00:00",
	"tts": false,
	"type": 0
}`

const testInteractionCreate_Slash = `{
	"application_id": "745036422834028565",
	"channel_id": "605598627141910556",
	"data": {
		"id": "891773437335462049",
		"name": "profile",
		"options": [
			{
				"name": "user",
				"type": 6,
				"value": "132715550571888640"
			}
		],
		"resolved": {
			"members": {
				"132715550571888640": {
					"avatar": null,
					"is_pending": false,
					"joined_at": "2016-03-31T03:17:39.375000+00:00",
					"nick": null,
					"pending": false,
					"permissions": "1099511627775",
					"premium_since": null,
					"roles": [
						"876685379770646538"
					]
				}
			},
			"users": {
				"132715550571888640": {
					"avatar": "1963eacbf364164efce1c597dc66aeab",
					"discriminator": "3719",
					"id": "132715550571888640",
					"public_flags": 0,
					"username": "bvisness"
				}
			}
		},
		"type": 1
	},
	"guild_id": "164936220651028480",
	"id": "891863243960750120",
	"member": {
		"avatar": null,
		"deaf": false,
		"is_pending": false,
		"joined_at": "2016-03-31T03:17:39.375000+00:00",
		"mute": false,
		"nick": null,
		"pending": false,
		"permissions": "1099511627775",
		"premium_since": null,
		"roles": [
			"876685379770646538"
		],
		"user": {
			"avatar": "1963eacbf364164efce1c597dc66aeab",
			"discriminator": "3719",
			"id": "132715550571888640",
			"public_flags": 0,
			"username": "bvisness"
		}
	},
	"token": "<redacted>",
	"type": 2,
	"version": 1
}`

const testInteractionCreate_User = `{
	"application_id": "745036422834028565",
	"channel_id": "605598627141910556",
	"data": {
		"id": "891856083914752031",
		"name": "HMN Profile",
		"resolved": {
			"members": {
				"132715550571888640": {
					"avatar": null,
					"is_pending": false,
					"joined_at": "2016-03-31T03:17:39.375000+00:00",
					"nick": null,
					"pending": false,
					"permissions": "1099511627775",
					"premium_since": null,
					"roles": [
						"876685379770646538"
					]
				}
			},
			"users": {
				"132715550571888640": {
					"avatar": "1963eacbf364164efce1c597dc66aeab",
					"discriminator": "3719",
					"id": "132715550571888640",
					"public_flags": 0,
					"username": "bvisness"
				}
			}
		},
		"target_id": "132715550571888640",
		"type": 2
	},
	"guild_id": "164936220651028480",
	"id": "891859116543324211",
	"member": {
		"avatar": null,
		"deaf": false,
		"is_pending": false,
		"joined_at": "2016-03-31T03:17:39.375000+00:00",
		"mute": false,
		"nick": null,
		"pending": false,
		"permissions": "1099511627775",
		"premium_since": null,
		"roles": [
			"876685379770646538"
		],
		"user": {
			"avatar": "1963eacbf364164efce1c597dc66aeab",
			"discriminator": "3719",
			"id": "132715550571888640",
			"public_flags": 0,
			"username": "bvisness"
		}
	},
	"token": "<redacted>",
	"type": 2,
	"version": 1
}`

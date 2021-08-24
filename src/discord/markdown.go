package discord

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"

	"git.handmade.network/hmn/hmn/src/config"
	"git.handmade.network/hmn/hmn/src/logging"
)

var (
	REMarkdownUser         = regexp.MustCompile(`<@([0-9]+)>`)
	REMarkdownUserNickname = regexp.MustCompile(`<@!([0-9]+)>`)
	REMarkdownChannel      = regexp.MustCompile(`<#([0-9]+)>`)
	REMarkdownRole         = regexp.MustCompile(`<@&([0-9]+)>`)
	REMarkdownCustomEmoji  = regexp.MustCompile(`<a?:(\w+):[0-9]+>`) // includes animated
	REMarkdownTimestamp    = regexp.MustCompile(`<t:([0-9]+)(:([tTdDfFR]))?>`)
)

func CleanUpMarkdown(ctx context.Context, original string) string {
	userMatches := REMarkdownUser.FindAllStringSubmatch(original, -1)
	userNicknameMatches := REMarkdownUserNickname.FindAllStringSubmatch(original, -1)
	channelMatches := REMarkdownChannel.FindAllStringSubmatch(original, -1)
	roleMatches := REMarkdownRole.FindAllStringSubmatch(original, -1)
	customEmojiMatches := REMarkdownCustomEmoji.FindAllStringSubmatch(original, -1)
	timestampMatches := REMarkdownTimestamp.FindAllStringSubmatch(original, -1)

	userIdsToFetch := map[string]struct{}{}

	for _, m := range userMatches {
		userIdsToFetch[m[1]] = struct{}{}
	}
	for _, m := range userNicknameMatches {
		userIdsToFetch[m[1]] = struct{}{}
	}

	// do the requests, gathering the resulting data
	userNames := map[string]string{}
	userNicknames := map[string]string{}
	channelNames := map[string]string{}
	roleNames := map[string]string{}
	var wg sync.WaitGroup
	var mutex sync.Mutex

	for userId := range userIdsToFetch {
		wg.Add(1)
		go func(ctx context.Context, userId string) {
			defer wg.Done()
			member, err := GetGuildMember(ctx, config.Config.Discord.GuildID, userId)
			if err != nil {
				if errors.Is(err, NotFound) {
					// not a problem
				} else if err != nil {
					logging.ExtractLogger(ctx).Warn().Err(err).Msg("failed to fetch guild member for markdown")
				}
				return
			}
			func() {
				mutex.Lock()
				defer mutex.Unlock()
				if member.User != nil {
					userNames[userId] = member.User.Username
				}
				if member.Nick != nil {
					userNicknames[userId] = *member.Nick
				}
			}()
		}(ctx, userId)
	}

	if len(channelMatches) > 0 {
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			channels, err := GetGuildChannels(ctx, config.Config.Discord.GuildID)
			if err != nil {
				logging.ExtractLogger(ctx).Warn().Err(err).Msg("failed to fetch channels for markdown")
				return
			}
			for _, channel := range channels {
				channelNames[channel.ID] = channel.Name
			}
		}(ctx)
	}

	if len(roleMatches) > 0 {
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()
			roles, err := GetGuildRoles(ctx, config.Config.Discord.GuildID)
			if err != nil {
				logging.ExtractLogger(ctx).Warn().Err(err).Msg("failed to fetch roles for markdown")
				return
			}
			for _, role := range roles {
				roleNames[role.ID] = role.Name
			}
		}(ctx)
	}

	wg.Wait()

	// Replace all the everything
	res := original
	for _, m := range userMatches {
		resultName := "Unknown User"
		if name, ok := userNames[m[1]]; ok {
			resultName = name
		}
		res = strings.Replace(res, m[0], fmt.Sprintf("@%s", resultName), 1)
	}
	for _, m := range userNicknameMatches {
		resultName := "Unknown User"
		if name, ok := userNicknames[m[1]]; ok {
			resultName = name
		} else if name, ok := userNames[m[1]]; ok {
			resultName = name
		}
		res = strings.Replace(res, m[0], fmt.Sprintf("@%s", resultName), 1)
	}
	for _, m := range channelMatches {
		resultName := "Unknown Channel"
		if name, ok := channelNames[m[1]]; ok {
			resultName = name
		}
		res = strings.Replace(res, m[0], fmt.Sprintf("#%s", resultName), 1)
	}
	for _, m := range roleMatches {
		resultName := "Unknown Role"
		if name, ok := roleNames[m[1]]; ok {
			resultName = name
		}
		res = strings.Replace(res, m[0], fmt.Sprintf("@%s", resultName), 1)
	}
	for _, m := range customEmojiMatches {
		res = strings.Replace(res, m[0], fmt.Sprintf(":%s:", m[1]), 1)
	}
	for _, m := range timestampMatches {
		res = strings.Replace(res, m[0], "<timestamp>", 1) // TODO: Actual timestamp stuff? Is it worth it?
	}

	return res
}

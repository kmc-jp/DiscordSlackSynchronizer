package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/kmc-jp/DiscordSlackSynchronizer/discord_webhook"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_emoji_block_maker"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_webhook"
	"github.com/pkg/errors"
)

type DiscordReactionHandler struct {
	discordHook *discord_webhook.Handler
	slackHook   *slack_webhook.Handler

	settings *SettingsHandler
}

func NewDiscordReactionHandler(slackHook *slack_webhook.Handler, discordHook *discord_webhook.Handler, settings *SettingsHandler) *DiscordReactionHandler {
	return &DiscordReactionHandler{
		slackHook:   slackHook,
		discordHook: discordHook,
		settings:    settings,
	}
}

func (d DiscordReactionHandler) GetReaction(guildID, channelID, messageID string) error {
	var sdt = d.settings.FindSlackChannel(channelID, guildID)
	if sdt.SlackChannel == "" {
		return nil
	}

	//Confirm Discord to Slack
	if !sdt.Setting.DiscordToSlack {
		return nil
	}

	message, err := d.discordHook.GetMessage(channelID, messageID)
	if err != nil {
		return errors.Wrap(err, "GetDiscordMessage")
	}

	srcMessages, err := d.slackHook.GetMessages(sdt.SlackChannel, "", 100)
	if err != nil {
		return errors.Wrap(err, "GetSlackMessages")
	}

	var srcMessage slack_webhook.Message
	var check bool

	dTime, err := message.Timestamp.Parse()
	if err != nil {
		return errors.Wrap(err, "ParseDiscordTS")
	}

	for i, msg := range srcMessages {
		if strings.Contains(msg.Text, "<"+SlackMessageDummyURI) {
			var sepMessage = strings.Split(msg.Text, "<"+SlackMessageDummyURI)
			var messageTS = strings.Split(sepMessage[len(sepMessage)-1], "|")[0]

			srcT, err := time.Parse(time.RFC3339, messageTS)
			if err != nil {
				continue
			}

			if dTime.UnixMilli() >= srcT.UnixMilli() {
				srcMessage = srcMessages[i]
				check = true
				break
			}
		}
	}
	if !check {
		return fmt.Errorf("MessageNotFound")
	}

	var blocks = slack_emoji_block_maker.Build(message.Reactions)

	for _, block := range srcMessage.Blocks {
		switch block.Type {
		case "image", "file":
			blocks = append([]slack_webhook.BlockBase{block}, blocks...)
		}
	}

	// add Slack text block if the message has text
	if strings.TrimSpace(strings.Split(srcMessage.Text, "<"+SlackMessageDummyURI)[0]) != "" {
		var element = slack_webhook.MrkdwnElement(srcMessage.Text)
		var textBlock = slack_webhook.ContextBlock(element)

		blocks = append([]slack_webhook.BlockBase{textBlock}, blocks...)
	}

	srcMessage.Blocks = blocks
	srcMessage.Channel = sdt.SlackChannel

	_, err = d.slackHook.Update(srcMessage)
	if err != nil {
		return errors.Wrap(err, "UpdateMessage")
	}

	return nil
}

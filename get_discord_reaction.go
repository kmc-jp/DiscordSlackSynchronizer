package main

import (
	"strings"

	"github.com/kmc-jp/DiscordSlackSynchronizer/discord_webhook"
	"github.com/kmc-jp/DiscordSlackSynchronizer/settings"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_emoji_block_maker"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_webhook"
	"github.com/pkg/errors"
)

type DiscordReactionHandler struct {
	discordHook *discord_webhook.Handler
	slackHook   *slack_webhook.Handler

	messageFinder *MessageFinder

	settings *settings.Handler
}

func NewDiscordReactionHandler(slackHook *slack_webhook.Handler, discordHook *discord_webhook.Handler, messageFinder *MessageFinder, settings *settings.Handler) *DiscordReactionHandler {
	return &DiscordReactionHandler{
		slackHook:     slackHook,
		discordHook:   discordHook,
		messageFinder: messageFinder,
		settings:      settings,
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

	srcMessage, meta, err := d.messageFinder.FindFromDiscordMessage(message, sdt.SlackChannel)
	if err != nil {
		return errors.Wrap(err, "FindFromDiscordMessage")
	}

	var originalMessage = meta.OriginalMessage

	var blocks = slack_emoji_block_maker.Build(message.Reactions)
	var check = false
	for _, block := range srcMessage.Blocks {
		switch block.Type {
		case "image", "file":
			blocks = append([]slack_webhook.BlockBase{block}, blocks...)
		case "section":
			if check {
				continue
			}
			originalMessage = block.Text.Text
			check = true
		}
	}

	// add Slack text block if the message has text
	if strings.TrimSpace(strings.Split(srcMessage.Text, "<"+SlackMessageDummyURI)[0]) != "" {
		var section = slack_webhook.SectionBlock()
		section.Text = slack_webhook.MrkdwnElement(originalMessage, false)

		blocks = append([]slack_webhook.BlockBase{section}, blocks...)
	}

	srcMessage.Blocks = blocks
	srcMessage.Channel = sdt.SlackChannel

	_, err = d.slackHook.Update(*srcMessage)
	if err != nil {
		return errors.Wrap(err, "UpdateMessage")
	}

	return nil
}

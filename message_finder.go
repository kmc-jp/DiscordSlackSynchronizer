package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/discord_webhook"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_webhook"
	"github.com/pkg/errors"
)

type MessageFinder struct {
	slackHook   *slack_webhook.Handler
	discordHook *discord_webhook.Handler

	escaper MessageEscaper
}

type MetaData struct {
	OriginalMessage string
	TS              string
	SlackUserID     string
	DiscordUserID   string
}

func NewMessageFinder(slackHook *slack_webhook.Handler, discordHook *discord_webhook.Handler) *MessageFinder {
	return &MessageFinder{
		discordHook: discordHook,
		slackHook:   slackHook,
	}
}

func (h *MessageFinder) SetMessageEscaper(escaper MessageEscaper) {
	h.escaper = escaper
}

func (h MessageFinder) ParseMetaData(text string) *MetaData {
	var meta MetaData

	var sepMessage = strings.Split(text, "<"+SlackMessageDummyURI)
	var uriQuery = strings.Split(sepMessage[len(sepMessage)-1], "|")[0]
	meta.TS = strings.Split(uriQuery, "&")[0]

	if len(sepMessage) == 1 {
		return nil
	}

	meta.OriginalMessage = strings.Join(sepMessage[:len(sepMessage)-1], "<"+SlackMessageDummyURI)

	if !strings.Contains(sepMessage[1], "&amp;slack_user_id=") {
		return &meta
	}

	uriQuery = strings.Split(uriQuery, "&amp;slack_user_id=")[1]
	meta.SlackUserID = strings.Split(uriQuery, "&amp;")[0]

	if !strings.Contains(uriQuery, "&amp;discord_user_id=") {
		return &meta
	}
	meta.DiscordUserID = strings.Split(uriQuery, "&amp;discord_user_id=")[0]

	return &meta
}

func (h MessageFinder) CreateURIformat(timestamp, slackUserID, discordUserID string) string {
	return fmt.Sprintf("<%s%s&slack_user_id=%s&discord_user_id=%s|%s>", SlackMessageDummyURI, timestamp, slackUserID, discordUserID, "ㅤ")
}

func (h MessageFinder) FindFromSlackMessage(srcContent *slack_webhook.Message, discordChannel string) (*discordgo.Message, *MetaData, error) {
	var message discordgo.Message
	var meta = h.ParseMetaData(srcContent.Text)
	if meta.TS != "" {
		messages, err := h.discordHook.GetMessages(discordChannel, "")
		if err != nil {
			return nil, nil, err
		}

		srcT, err := time.Parse(time.RFC3339, meta.TS)
		if err != nil {
			goto next
		}

		for i, msg := range messages {
			if i == 0 {
				continue
			}

			if msg.Timestamp.UnixMilli() < srcT.UnixMilli() {
				message = messages[i-1]
				break
			}
		}
	}

next:
	// if message not found, find by its message text
	if message.ID == "" {
		messages, err := h.discordHook.GetMessages(discordChannel, "")
		if err != nil {
			return nil, nil, err
		}

		content, err := h.escaper.EscapeMessage(srcContent.Text)
		if err != nil {
			return nil, nil, err
		}

		for i, msg := range messages {
			if content == msg.Content {
				message = messages[i-1]
				break
			}
		}
	}

	// not found
	if message.ID == "" {
		return nil, nil, fmt.Errorf("MessageNotFound")
	}

	return &message, meta, nil
}

func (h MessageFinder) FindFromDiscordMessage(message discordgo.Message, slackChannel string) (*slack_webhook.Message, *MetaData, error) {
	srcMessages, err := h.slackHook.GetMessages(slackChannel, "", 100)
	if err != nil {
		return nil, nil, errors.Wrap(err, "GetSlackMessages")
	}

	var srcMessage slack_webhook.Message
	var check bool

	dTime := message.Timestamp

	var meta *MetaData

	for i, msg := range srcMessages {
		if strings.Contains(msg.Text, "<"+SlackMessageDummyURI) {
			meta = h.ParseMetaData(msg.Text)
			var messageTS = meta.TS

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
		return nil, nil, errors.New("NotFound")
	}

	return &srcMessage, meta, nil
}

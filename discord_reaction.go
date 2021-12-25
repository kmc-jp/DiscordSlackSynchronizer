package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/discord_webhook"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_emoji_imager"
)

type DiscordReactionHandler struct {
	token string

	reactionImager ReactionImagerType

	hook *discord_webhook.Handler

	slack   MessageGetter
	escaper MessageEscaper
}

type ReactionImagerType interface {
	AddEmoji(name string, uri string)
	RemoveEmoji(string)
	MakeReactionsImage(channel string, timestamp string) (r io.Reader, err error)
	GetEmojiURI(name string) string
}

func NewDiscordReactionHandler(token string) *DiscordReactionHandler {
	return &DiscordReactionHandler{
		token: token,
	}
}

func (d *DiscordReactionHandler) SetReactionImager(imager ReactionImagerType) {
	d.reactionImager = imager
}

func (d *DiscordReactionHandler) SetMessageEscaper(escaper MessageEscaper) {
	d.escaper = escaper
}

func (d *DiscordReactionHandler) SetMessageGetter(getter MessageGetter) {
	d.slack = getter
}

func (d *DiscordReactionHandler) SetDiscordWebhook(hook *discord_webhook.Handler) {
	d.hook = hook
}

func (d *DiscordReactionHandler) GetReaction(channel string, timestamp string) error {
	const ReactionGifName = "reactions.gif"

	var zeroReaction bool

	r, err := d.reactionImager.MakeReactionsImage(channel, timestamp)
	switch err {
	case nil:
		break
	case slack_emoji_imager.ErrorNoReactions:
		zeroReaction = true
	default:
		return err
	}

	var cs, _ = findDiscordChannel(channel)
	if !cs.Setting.SlackToDiscord {
		return nil
	}

	srcContent, err := d.slack.GetMessage(channel, timestamp)
	if err != nil {
		return err
	}

	var message discord_webhook.Message
	if strings.Contains(srcContent, "<"+SlackMessageDummyURI) {
		var sepMessage = strings.Split(srcContent, "<"+SlackMessageDummyURI)
		var messageTS = strings.Split(sepMessage[len(sepMessage)-1], "|")[0]

		messages, err := d.getMessages(cs.DiscordChannel, "")
		if err != nil {
			return err
		}

		srcT, err := time.Parse(time.RFC3339, messageTS)
		if err != nil {
			goto next
		}

		for i, msg := range messages {
			if i == 0 {
				continue
			}
			t, err := msg.Timestamp.Parse()
			if err != nil {
				goto next
			}

			if t.UnixMilli() < srcT.UnixMilli() {
				message.Message = &messages[i-1]
				break
			}
		}
	}

next:
	if message.Message == nil {
		messages, err := d.getMessages(cs.DiscordChannel, "")
		if err != nil {
			return err
		}

		srcContent, err = d.escaper.EscapeMessage(srcContent)
		if err != nil {
			return err
		}

		for i, msg := range messages {
			if srcContent == msg.Content {
				message.Message = &messages[i]
				break
			}
		}
	}
	if message.Message == nil {
		return fmt.Errorf("MessageNotFound")
	}

	message.Attachments = make([]discord_webhook.Attachment, 0)

	for i := range message.Message.Attachments {
		if message.Message.Attachments[i] == nil {
			continue
		}

		var oldAtt = message.Message.Attachments[i]
		if strings.HasSuffix(oldAtt.URL, ReactionGifName) {
			continue
		}

		id, err := strconv.Atoi(oldAtt.ID)
		if err != nil {
			continue
		}

		message.Attachments = append(message.Attachments, discord_webhook.Attachment{
			URL:      oldAtt.URL,
			ID:       id,
			ProxyURL: oldAtt.ProxyURL,
			Filename: oldAtt.Filename,
			Width:    oldAtt.Width,
			Height:   oldAtt.Height,
			Size:     oldAtt.Size,
		})

	}

	if zeroReaction {
		return d.hook.Edit(message.ChannelID, message.ID, message, []discord_webhook.File{})
	}

	var file = discord_webhook.File{
		FileName:    ReactionGifName,
		Reader:      r,
		ContentType: "image/gif",
	}

	return d.hook.Edit(message.ChannelID, message.ID, message, []discord_webhook.File{file})
}

func (d *DiscordReactionHandler) getMessage(channelID, messageID string) (messages discordgo.Message, err error) {
	var requestAttr = make(url.Values)

	var client = http.DefaultClient
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/channels/%s/messages/%s?%s", DiscordAPIEndpoint, channelID, messageID, requestAttr.Encode()),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+d.token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var responseAttr discordgo.Message
	err = func() error {
		defer func() {
			err := recover()
			if err != nil {
				log.Println(err)
			}
		}()
		return json.NewDecoder(resp.Body).Decode(&responseAttr)
	}()
	if err != nil {
		return
	}

	return responseAttr, nil
}

func (d *DiscordReactionHandler) getMessages(channelID string, around string) (messages []discordgo.Message, err error) {
	var requestAttr = make(url.Values)

	requestAttr.Set("limit", "100")
	if around != "" {
		requestAttr.Set("around", around)
	}

	var client = http.DefaultClient
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/channels/%s/messages?%s", DiscordAPIEndpoint, channelID, requestAttr.Encode()),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+d.token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var responseAttr []discordgo.Message
	err = func() error {
		defer func() {
			err := recover()
			if err != nil {
				log.Println(err)
				buf := new(bytes.Buffer)
				io.Copy(buf, resp.Body)
				fmt.Println(buf)
			}
		}()
		return json.NewDecoder(resp.Body).Decode(&responseAttr)
	}()
	if err != nil {
		return
	}

	return responseAttr, nil
}

func (d *DiscordReactionHandler) AddEmoji(name, value string) {
	d.reactionImager.AddEmoji(name, value)
}

func (d *DiscordReactionHandler) RemoveEmoji(name string) {
	d.reactionImager.RemoveEmoji(name)
}

func (d *DiscordReactionHandler) GetEmojiURI(name string) string {
	return d.reactionImager.GetEmojiURI(name)
}

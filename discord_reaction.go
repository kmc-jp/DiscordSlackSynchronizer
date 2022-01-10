package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/discord_webhook"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_emoji_imager"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_webhook"
	"github.com/pkg/errors"
)

type DiscordReactionHandler struct {
	token string

	reactionImager ReactionImagerType

	hook      *discord_webhook.Handler
	slackHook *slack_webhook.Handler

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

func (d *DiscordReactionHandler) SetDiscordWebhook(hook *discord_webhook.Handler) {
	d.hook = hook
}

func (d *DiscordReactionHandler) SetSlackWebhook(hook *slack_webhook.Handler) {
	d.slackHook = hook
}

func (d *DiscordReactionHandler) GetReaction(channel string, timestamp string) error {
	const ReactionGifName = "reactions.gif"

	var cs, _ = findDiscordChannel(channel)
	if !cs.Setting.SlackToDiscord {
		return nil
	}

	srcContent, err := d.slackHook.GetMessage(channel, timestamp)
	if err != nil {
		return errors.Wrap(err, "SlackGetMessage")
	}
	srcContent.Channel = channel

	var oldAttachments []*discordgo.MessageAttachment
	var message discord_webhook.Message
	if strings.Contains(srcContent.Text, "<"+SlackMessageDummyURI) {
		var sepMessage = strings.Split(srcContent.Text, "<"+SlackMessageDummyURI)
		var messageTS = strings.Split(sepMessage[len(sepMessage)-1], "|")[0]

		messages, err := d.hook.GetMessages(cs.DiscordChannel, "")
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
				oldAttachments = messages[i-1].Attachments
				break
			}
		}
	}

next:
	// if message not found, find by its message text
	if message.Message == nil {
		messages, err := d.hook.GetMessages(cs.DiscordChannel, "")
		if err != nil {
			return err
		}

		content, err := d.escaper.EscapeMessage(srcContent.Text)
		if err != nil {
			return err
		}

		for i, msg := range messages {
			if content == msg.Content {

				message.Message = &messages[i]
				break
			}
		}
	}

	// not found
	if message.Message == nil {
		return fmt.Errorf("MessageNotFound")
	}

	var dFiles = []discord_webhook.File{}

	message.Attachments = make([]discord_webhook.Attachment, 0)

	for i := range message.Message.Attachments {
		if message.Message.Attachments[i] == nil {
			continue
		}

		var attach = message.Message.Attachments[i]
		if attach.Filename == ReactionGifName {
			// Reaction Gif should be renewed
			continue
		}

		resp, err := http.Get(attach.URL)
		if err != nil {
			log.Printf("GetFileError: %s\n", err.Error())
			continue
		}
		defer resp.Body.Close()

		var dFile = discord_webhook.File{
			Reader:      resp.Body,
			FileName:    attach.Filename,
			ContentType: discord_webhook.FindContentType(attach.Filename),
		}

		dFiles = append(dFiles, dFile)
	}

	r, err := d.reactionImager.MakeReactionsImage(channel, timestamp)
	switch err {
	case nil:
		break
	case slack_emoji_imager.ErrorNoReactions:
		_, err = d.hook.Edit(message.ChannelID, message.ID, message, dFiles)
		return errors.Wrap(err, "EditSlackMessage")
	default:
		return errors.Wrap(err, "MakeReactionImage")
	}

	var file = discord_webhook.File{
		FileName:    ReactionGifName,
		Reader:      r,
		ContentType: "image/gif",
	}

	newMessage, err := d.hook.Edit(message.ChannelID, message.ID, message, append(dFiles, file))
	if err != nil {
		return errors.Wrap(err, "DiscordMessageEdit")
	}

	for i, block := range srcContent.Blocks {
		switch block.Type {
		case "image":
			for j, oldAttachment := range oldAttachments {
				if len(newMessage.Attachments) <= j || newMessage.Attachments[j].Filename != oldAttachment.Filename {
					fmt.Println(newMessage.Attachments[j].Filename, oldAttachment.Filename)
					continue
				}
				srcContent.Blocks[i] = slack_webhook.ImageBlock(newMessage.Attachments[j].URL, block.AltText)
			}
		case "file":
			for j, oldAttachment := range oldAttachments {
				var externalID = fmt.Sprintf("%s:%s/%s", ProgramName, newMessage.ChannelID, oldAttachment.ID)

				if block.ExternalID != externalID && len(newMessage.Attachments) <= j {
					continue
				}

				sFile, err := d.slackHook.FilesRemoteInfo(externalID, "")
				if err != nil {
					log.Printf("FilesRemoteInfo: %s\n", err.Error())
					continue
				}

				externalID = fmt.Sprintf("%s:%s/%s", ProgramName, newMessage.ChannelID, newMessage.Attachments[j].ID)

				_, err = d.slackHook.FilesRemoteAdd(
					slack_webhook.FilesRemoteAddParameters{
						ExternalURL: newMessage.Attachments[j].URL,
						Title:       sFile.Title,
						FileType:    sFile.Filetype,
						ExternalID:  externalID,
					},
				)

				if err != nil {
					log.Printf("FilesRemoteAdd: %s\n", err.Error())
					continue
				}

				srcContent.Blocks[i].ExternalID = externalID
			}
		}
	}

	_, err = d.slackHook.Update(*srcContent)

	return errors.Wrap(err, "UpdateSlackMessage")
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

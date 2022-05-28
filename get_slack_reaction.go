package main

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/discord_webhook"
	"github.com/kmc-jp/DiscordSlackSynchronizer/settings"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_emoji_imager"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_webhook"
	"github.com/pkg/errors"
)

type SlackReactionHandler struct {
	reactionImager ReactionImagerType

	discordHook *discord_webhook.Handler
	slackHook   *slack_webhook.Handler

	messageFinder *MessageFinder

	settings *settings.Handler

	escaper MessageEscaper
}

type ReactionImagerType interface {
	AddEmoji(name string, uri string)
	RemoveEmoji(string)
	MakeReactionsImage(channel string, timestamp string) (r io.Reader, err error)
	GetEmojiURI(name string) string
}

func NewSlackReactionHandler(slackHook *slack_webhook.Handler, discordHook *discord_webhook.Handler, messageFinder *MessageFinder, settings *settings.Handler) *SlackReactionHandler {
	return &SlackReactionHandler{
		slackHook:     slackHook,
		discordHook:   discordHook,
		messageFinder: messageFinder,
		settings:      settings,
	}
}

func (d *SlackReactionHandler) SetReactionImager(imager ReactionImagerType) {
	d.reactionImager = imager
}

func (d *SlackReactionHandler) SetMessageEscaper(escaper MessageEscaper) {
	d.escaper = escaper
}

func (d *SlackReactionHandler) GetReaction(channel string, timestamp string) error {
	const ReactionGifName = "reactions.gif"

	var cs, _ = d.settings.FindDiscordChannel(channel)
	if !cs.Setting.SlackToDiscord {
		return nil
	}

	srcContent, err := d.slackHook.GetMessage(channel, timestamp)
	if err != nil {
		return errors.Wrap(err, "SlackGetMessage")
	}
	srcContent.Channel = channel

	var oldAttachments []*discordgo.MessageAttachment
	dMessage, _, err := d.messageFinder.FindFromSlackMessage(srcContent, cs.DiscordChannel)
	if err != nil {
		return errors.Wrap(err, "FindFromSlackMessage")
	}

	oldAttachments = dMessage.Attachments
	var message = discord_webhook.FromDiscordgoMessage(dMessage)

	// not found
	if message.ID == "" {
		return fmt.Errorf("MessageNotFound")
	}

	var dFiles = []discord_webhook.File{}

	for _, attach := range message.Attachments {
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

	message.Attachments = []discord_webhook.Attachment{}

	r, err := d.reactionImager.MakeReactionsImage(channel, timestamp)
	switch err {
	case nil:
		dFiles = append(
			dFiles,
			discord_webhook.File{
				FileName:    ReactionGifName,
				Reader:      r,
				ContentType: "image/gif",
			},
		)
	case slack_emoji_imager.ErrorNoReactions:

	default:
		return errors.Wrap(err, "MakeReactionImage")
	}

	newMessage, err := d.discordHook.Edit(message.ChannelID, message.ID, message, dFiles)
	if err != nil {
		return errors.Wrap(err, "DiscordMessageEdit")
	}

	// reset file and image urls of Slack blocks
	for i, block := range srcContent.Blocks {
		switch block.Type {
		case "image":
			var attachmentIndex int
			var found bool
			for j, oldAttachment := range oldAttachments {
				// find attachment of the image
				if len(newMessage.Attachments) <= j || newMessage.Attachments[j].Filename != oldAttachment.Filename {
					continue
				}
				if oldAttachment.URL != block.ImageURL {
					continue
				}
				found = true
				attachmentIndex = j
				break
			}
			if !found {
				continue
			}

			srcContent.Blocks[i] = slack_webhook.ImageBlock(newMessage.Attachments[attachmentIndex].URL, block.AltText)
			srcContent.Blocks[i].Title = slack_webhook.ImageTitle(block.Title.Text, false)
		case "file":
			var attachmentIndex int
			var externalID string
			var found bool

			// find attachment of the file
			for j, oldAttachment := range oldAttachments {
				externalID = fmt.Sprintf("%s:%s/%s", ProgramName, newMessage.ChannelID, oldAttachment.ID)
				if block.ExternalID != externalID || len(newMessage.Attachments) <= j {
					continue
				}

				attachmentIndex = j
				found = true

				break
			}
			if !found {
				continue
			}

			sFile, err := d.slackHook.FilesRemoteInfo(externalID, "")
			if err != nil {
				log.Printf("FilesRemoteInfo: %s\n", err.Error())
				continue
			}

			err = d.slackHook.FilesRemoteRemove(externalID, "")
			if err != nil {
				log.Printf("FilesRemoteRemove: %s\n", err.Error())
			}

			externalID = fmt.Sprintf("%s:%s/%s", ProgramName, newMessage.ChannelID, newMessage.Attachments[attachmentIndex].ID)

			_, err = d.slackHook.FilesRemoteAdd(
				slack_webhook.FilesRemoteAddParameters{
					ExternalURL: newMessage.Attachments[attachmentIndex].URL,
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

	_, err = d.slackHook.Update(*srcContent)

	return errors.Wrap(err, "UpdateSlackMessage")
}

func (d *SlackReactionHandler) AddEmoji(name, value string) {
	d.reactionImager.AddEmoji(name, value)
}

func (d *SlackReactionHandler) RemoveEmoji(name string) {
	d.reactionImager.RemoveEmoji(name)
}

func (d *SlackReactionHandler) GetEmojiURI(name string) string {
	return d.reactionImager.GetEmojiURI(name)
}

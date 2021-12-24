package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_emoji_imager"
)

type DiscordReactionHandler struct {
	token string

	reactionImager ReactionImagerType

	slack   MessageGetter
	escaper MessageEscaper
}

type ReactionImagerType interface {
	AddEmoji(name string, uri string)
	RemoveEmoji(string)
	MakeReactionsImage(channel string, timestamp string) (r io.Reader, err error)
	GetEmojiURI(name string) string
}

type DiscordMessage struct {
	*discordgo.Message
	Components  []DiscordComponent  `json:"components,omitempty"`
	AvaterURL   string              `json:"avatar_url,omitempty"`
	Attachments []DiscordAttachment `json:"attachments"`
	UserName    string              `json:"username,omitempty"`
}

type DiscordComponent struct {
	Type        int                `json:"type"`
	CustomID    string             `json:"custom_id,omitempty"`
	Disabled    bool               `json:"disabled,omitempty"`
	Style       int                `json:"style,omitempty"`
	Label       string             `json:"label,omitempty"`
	URL         string             `json:"url,omitempty"`
	Placeholder string             `json:"placeholder,omitempty"`
	MinValues   int                `json:"min_values,omitempty"`
	MaxValues   int                `json:"max_values,omitempty"`
	Components  []DiscordComponent `json:"components,omitempty"`
	Emoji       *discordgo.Emoji   `json:"emoji,omitempty"`
	Options     []DiscordOption    `json:"options,omitempty"`
}

type DiscordOption struct {
	Label       string           `json:"label"`
	Value       string           `json:"value"`
	Description string           `json:"description,omitempty"`
	Emoji       *discordgo.Emoji `json:"emoji,omitempty"`
	Default     bool             `json:"default,omitempty"`
}

type DiscordAttachment struct {
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	ID          int    `json:"id"`
	ProxyURL    string `json:"proxy_url,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Size        int    `json:"size,omitempty"`
}

type DiscordFile struct {
	FileName    string
	Reader      io.Reader
	ContentType string
}

const DiscordReactionThreadName = "SlackEmoji"

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

	messages, err := d.getMessages(cs.DiscordChannel, timestamp)
	if err != nil {
		return err
	}

	srcContent, err := d.slack.GetMessage(channel, timestamp)
	if err != nil {
		return err
	}

	srcContent, err = d.escaper.EscapeMessage(srcContent)
	if err != nil {
		return err
	}

	var message DiscordMessage
	for i, msg := range messages {
		if srcContent == msg.Content {
			message.Message = &messages[i]
			break
		}
	}
	if message.Message == nil {
		return fmt.Errorf("MessageNotFound")
	}

	message.Attachments = make([]DiscordAttachment, 0)

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

		message.Attachments = append(message.Attachments, DiscordAttachment{
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
		return DiscordWebhook.Edit(message.ChannelID, message.ID, message, []DiscordFile{})
	}

	var file = DiscordFile{
		FileName:    ReactionGifName,
		Reader:      r,
		ContentType: "image/gif",
	}

	return DiscordWebhook.Edit(message.ChannelID, message.ID, message, []DiscordFile{file})
}

func (d *DiscordReactionHandler) getMessages(channelID, timestamp string) (messages []discordgo.Message, err error) {
	if !strings.Contains(timestamp, ".") {
		return
	}

	var requestAttr = make(url.Values)

	requestAttr.Set("limit", "100")

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

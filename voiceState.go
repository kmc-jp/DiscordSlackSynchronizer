package main

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

var voiceChannels DiscordVoiceChannels = DiscordVoiceChannels{
	map[string]*VoiceChannels{},
	sync.Mutex{},
}

type DiscordVoiceChannels struct {
	Guilds map[string]*VoiceChannels
	Mutex  sync.Mutex
}

type VoiceChannels struct {
	Channels map[string]*VoiceChannel
}

type VoiceChannel struct {
	Users   map[string]*VoiceState
	Channel *discordgo.Channel
}

type VoiceState struct {
	Muted    bool
	Deafened bool
	Member   *discordgo.Member
}

// Join returns already user exits
func (v *VoiceChannels) Join(channel *discordgo.Channel, member *discordgo.Member) bool {
	if v == nil || v.Channels == nil {
		v.Channels = map[string]*VoiceChannel{}
	}
	if _, ok := v.Channels[channel.ID]; !ok {
		v.Channels[channel.ID] = &VoiceChannel{
			Users:   map[string]*VoiceState{},
			Channel: channel,
		}
	}
	ch, ok := v.FindChannelHasUser(member.User.ID)
	exists := false
	if ok {
		if ch == channel.ID {
			exists = true
		} else {
			// User changed channels
			v.Leave(member.User.ID)
		}
	}
	v.Channels[channel.ID].Users[member.User.ID] = &VoiceState{
		Muted: false, Deafened: false, Member: member,
	}
	return exists
}

func (v *VoiceChannels) Leave(userID string) {
	if v == nil || v.Channels == nil {
		v.Channels = map[string]*VoiceChannel{}
	}
	for _, channel := range v.Channels {
		_, ok := channel.Users[userID]
		if ok {
			delete(channel.Users, userID)
		}
	}
}

func (v *VoiceChannels) FindChannelHasUser(userID string) (string, bool) {
	if v == nil || v.Channels == nil {
		v.Channels = map[string]*VoiceChannel{}
	}
	for _, channel := range v.Channels {
		_, ok := channel.Users[userID]
		if ok {
			return channel.Channel.ID, true
		}
	}
	return "", false
}

func (v *VoiceChannels) Muted(userID string) {
	if v == nil || v.Channels == nil {
		v.Channels = map[string]*VoiceChannel{}
	}
	for _, channel := range v.Channels {
		user, ok := channel.Users[userID]
		if ok {
			user.Muted = true
			user.Deafened = false
		}
	}
}

func (v *VoiceChannels) Deafened(userID string) {
	if v == nil || v.Channels == nil {
		v.Channels = map[string]*VoiceChannel{}
	}
	for _, channel := range v.Channels {
		user, ok := channel.Users[userID]
		if ok {
			user.Muted = true
			user.Deafened = true
		}
	}
}

func (v VoiceChannels) SlackBlocksMultiChannel() ([]json.RawMessage, error) {
	blocks := []json.RawMessage{}

	for _, channel := range v.Channels {
		if len(channel.Users) == 0 {
			// skip no user channels
			continue
		}
		channelBlocks, err := channel.SlackBlocksSingleChannel()
		if err != nil {
			return nil, errors.Wrapf(err, "Channel Slack Block")
		}
		blocks = append(blocks, channelBlocks...)
	}
	if len(blocks) <= 1 {
		element, err := MarkdownElement("誰もいない")
		if err != nil {
			return nil, errors.Wrapf(err, "Missing Markdown Element")
		}
		block, err := ContextBlock([]json.RawMessage{element})
		if err != nil {
			return nil, errors.Wrapf(err, "Context Element")
		}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (c VoiceChannel) SlackBlocksSingleChannel() ([]json.RawMessage, error) {
	var blocks = []json.RawMessage{}

	channelText := fmt.Sprintf("<https://discord.com/channels/%s|%s: >", c.Channel.GuildID, c.Channel.Name)
	channels, err := MarkdownElement(channelText)
	if err != nil {
		return nil, errors.Wrapf(err, "Channel Element")
	}

	block, err := ContextBlock([]json.RawMessage{channels})
	if err != nil {
		return nil, errors.Wrapf(err, "Channel Elements")
	}
	blocks = append(blocks, block)

	// Sort By User State
	normal := []*VoiceState{}
	muted := []*VoiceState{}
	deafened := []*VoiceState{}

	for _, user := range c.Users {
		if user.Deafened {
			deafened = append(deafened, user)
		} else if user.Muted {
			muted = append(muted, user)
		} else {
			normal = append(normal, user)
		}
	}

	users := append(normal, muted...)
	users = append(users, deafened...)

	var userCount int
	var elements = []json.RawMessage{}

	for _, user := range users {
		userImage := user.Member.User.AvatarURL("")
		username := user.Member.Nick
		if username == "" {
			username = user.Member.User.Username
		}

		imageElm, err := ImageElement(userImage, username)
		if err != nil {
			return nil, errors.Wrapf(err, "Avatar Element")
		}
		emoji := ""
		if user.Muted {
			emoji = ":discord_muted:"
		}
		if user.Deafened {
			emoji = ":discord_deafened:"
		}
		text := fmt.Sprintf("%s%s ", emoji, username)
		userElm, err := MarkdownElement(text)
		if err != nil {
			return nil, errors.Wrapf(err, "Username Element")
		}
		elements = append(elements, imageElm, userElm)

		userCount++
		if userCount%4 == 0 {
			block, err := ContextBlock(elements)
			if err != nil {
				return nil, errors.Wrapf(err, "Channel Elements")
			}
			blocks = append(blocks, block)

			elements = []json.RawMessage{}
		}
	}

	if userCount%4 > 0 {
		block, err = ContextBlock(elements)
		if err != nil {
			return nil, errors.Wrapf(err, "Channel Elements")
		}
		blocks = append(blocks, block)
	}

	div, err := DividerBlock()
	if err != nil {
		return nil, errors.Wrapf(err, "Divider Block")
	}
	blocks = append(blocks, div)

	return blocks, nil
}

func ContextBlock(elements []json.RawMessage) (json.RawMessage, error) {
	type block struct {
		Type     string            `json:"type"`
		Elements []json.RawMessage `json:"elements"`
	}
	res := block{Type: "context", Elements: elements}
	return json.Marshal(res)
}

func ImageElement(ImageURL string, AltText string) (json.RawMessage, error) {
	type element struct {
		Type     string `json:"type"`
		ImageURL string `json:"image_url"`
		AltText  string `json:"alt_text"`
	}
	res := element{Type: "image", ImageURL: ImageURL, AltText: AltText}
	return json.Marshal(res)
}

func MarkdownElement(text string) (json.RawMessage, error) {
	type element struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	res := element{Type: "mrkdwn", Text: text}
	return json.Marshal(res)
}

func DividerBlock() (json.RawMessage, error) {
	type block struct {
		Type string `json:"type"`
	}
	res := block{Type: "divider"}
	return json.Marshal(res)
}

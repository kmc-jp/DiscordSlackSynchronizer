package main

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_webhook"
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

func (v VoiceChannels) SlackBlocksMultiChannel() ([]slack_webhook.BlockBase, error) {
	var blocks = []slack_webhook.BlockBase{}

	for _, channel := range v.Channels {
		if len(channel.Users) == 0 {
			// skip no user channels
			continue
		}
		var channelBlocks = channel.SlackBlocksSingleChannel()

		blocks = append(blocks, channelBlocks...)
	}
	if len(blocks) <= 1 {
		element := slack_webhook.BlockElement{Type: "mrkdwn", Text: "誰もいない"}

		var block = slack_webhook.BlockBase{Type: "context", Elements: []slack_webhook.BlockElement{element}}
		blocks = append(blocks, block)
	}
	return blocks, nil
}

func (c VoiceChannel) SlackBlocksSingleChannel() []slack_webhook.BlockBase {
	var blocks = []slack_webhook.BlockBase{}

	channelText := fmt.Sprintf("<https://discord.com/channels/%s|%s: >", c.Channel.GuildID, c.Channel.Name)

	var channelNameElement = slack_webhook.BlockElement{Type: "mrkdwn", Text: channelText}

	blocks = append(
		blocks,
		slack_webhook.BlockBase{
			Type:     "context",
			Elements: []slack_webhook.BlockElement{channelNameElement},
		},
	)

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
	var elements = []slack_webhook.BlockElement{}

	for _, user := range users {
		userImage := user.Member.User.AvatarURL("")
		username := user.Member.Nick
		if username == "" {
			username = user.Member.User.Username
		}

		var imageElm = slack_webhook.BlockElement{
			Type:     "image",
			ImageURL: userImage,
			AltText:  username,
		}

		emoji := ""
		if user.Muted {
			emoji = ":discord_muted:"
		}
		if user.Deafened {
			emoji = ":discord_deafened:"
		}

		text := fmt.Sprintf("%s%s ", emoji, username)
		var userElm = slack_webhook.BlockElement{Type: "mrkdwn", Text: text}

		elements = append(elements, imageElm, userElm)

		userCount++
		if userCount%4 == 0 {
			var block = slack_webhook.BlockBase{
				Type:     "context",
				Elements: elements,
			}

			blocks = append(blocks, block)

			elements = []slack_webhook.BlockElement{}
		}
	}

	if userCount%4 > 0 {
		var block = slack_webhook.BlockBase{
			Type:     "context",
			Elements: elements,
		}

		blocks = append(blocks, block)
	}

	var div = slack_webhook.BlockBase{Type: "divider"}
	blocks = append(blocks, div)

	return blocks
}

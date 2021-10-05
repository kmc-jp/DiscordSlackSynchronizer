package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
)

type DiscordHandler struct {
	Session *discordgo.Session
	regExp  struct {
		UserID   *regexp.Regexp
		Channel  *regexp.Regexp
		ImageURI *regexp.Regexp
	}
}

func NewDiscordBot(apiToken string) *DiscordHandler {

	// Create a new Discord session using the provided bot token.
	dg, err := discordgo.New("Bot " + Tokens.Discord.API)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return nil
	}

	var d DiscordHandler

	d.Session = dg
	d.regExp.UserID = regexp.MustCompile(`<@!(\d+)>`)
	d.regExp.Channel = regexp.MustCompile(`<#(\d+)>`)
	d.regExp.ImageURI = regexp.MustCompile(`\S\.png|\.jpg|\.jpeg|\.gif`)

	dg.AddHandler(d.voiceState)
	dg.AddHandler(d.watch)

	return &d
}

func (d *DiscordHandler) Close() error {
	return d.Session.Close()
}

func (d *DiscordHandler) Do() error {
	return d.Session.Open()
}

func (d *DiscordHandler) watch(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID || m.Author.Bot {
		return
	}

	var sdt = findSlackChannel(m.ChannelID, m.GuildID)
	if sdt.SlackChannel == "" {
		return
	}

	//Confirm Discord to Slack
	if !sdt.Setting.DiscordToSlack {
		return
	}

	var imageURIs = []string{}
	var fileURL string
	for _, f := range m.Attachments {
		if d.regExp.ImageURI.MatchString(f.URL) {
			imageURIs = append(imageURIs, f.URL)
		} else {
			fileURL += "\n" + f.URL
		}
	}

	var name string = m.Member.Nick
	if name == "" {
		name = m.Author.Username
	}

	var content string

	content = m.Content

	for _, id := range d.regExp.UserID.FindAllStringSubmatch(content, -1) {
		if len(id) < 2 {
			continue
		}

		mem, err := s.GuildMember(m.GuildID, id[1])
		if err != nil {
			continue
		}

		var idName = mem.Nick
		if idName == "" {
			idName = mem.User.Username
		}
		content = strings.Join(strings.Split(content, "!"+id[1]), idName)
	}

	for _, ch := range d.regExp.Channel.FindAllStringSubmatch(content, -1) {
		if len(ch) < 2 {
			continue
		}

		channel, err := s.State.GuildChannel(m.GuildID, ch[1])
		if err != nil {
			continue
		}

		content = strings.Join(strings.Split(content,
			fmt.Sprintf("<#%s>", ch[1])),
			fmt.Sprintf(
				"<https://discord.com/channels/%s/%s|#%s>",
				m.GuildID, ch[1], channel.Name,
			),
		)
	}

	if sdt.Setting.ShowChannelName {
		channelData, err := s.State.GuildChannel(m.GuildID, m.ChannelID)
		if err != nil {
			fmt.Printf("%s\n", err.Error())
			return
		}
		content = "`#" + channelData.Name + "` " + content + fileURL
	} else {
		content = content + fileURL
	}

	var attachments = []SlackAttachment{}
	for _, imageURI := range imageURIs {
		var attachment SlackAttachment

		attachment.ImageURL = imageURI
		attachments = append(attachments, attachment)
	}

	// TODO: create channel if not exist option

	var message = SlackHookMessage{
		Channel:     sdt.SlackChannel,
		Name:        name,
		Text:        content,
		IconURL:     m.Author.AvatarURL(""),
		Attachments: attachments,
	}

	message.Send()
}

func (d *DiscordHandler) voiceState(s *discordgo.Session, vs *discordgo.VoiceStateUpdate) {
	voiceChannels.Mutex.Lock()
	defer voiceChannels.Mutex.Unlock()

	if vs.UserID == s.State.User.ID {
		return
	}

	channel, e := s.State.Channel(vs.VoiceState.ChannelID)

	if voiceChannels.Guilds[vs.GuildID] == nil {
		voiceChannels.Guilds[vs.GuildID] = &VoiceChannels{}
	}

	channels := voiceChannels.Guilds[vs.GuildID]

	if e != nil || vs.ChannelID == "" { // If the channel is missing, the user has left
		channel, ok := channels.FindChannelHasUser(vs.UserID)
		if !ok {
			return
		}
		channels.Leave(vs.UserID)
		setting := findSlackChannel(channel, vs.VoiceState.GuildID)
		d.sendVoiceState(setting, channels)

	} else { // User joind or State changed
		setting := findSlackChannel(vs.VoiceState.ChannelID, vs.VoiceState.GuildID)
		mem, err := s.GuildMember(vs.GuildID, vs.UserID)
		if err != nil {
			fmt.Printf("Failed to get info of a member: %v\n", err)
			return
		}
		exists := channels.Join(channel, mem)

		if vs.SelfDeaf {
			channels.Deafened(vs.UserID)
		} else if vs.Mute || vs.SelfMute {
			channels.Muted(vs.UserID)
		}

		if setting.Setting.SendMuteState || !exists {
			d.sendVoiceState(setting, channels)
		}
	}
}

func (d *DiscordHandler) sendVoiceState(setting ChannelSetting, channels *VoiceChannels) {
	if setting.SlackChannel == "" {
		return
	}
	if !setting.Setting.SendVoiceState {
		return
	}
	var blocks []json.RawMessage
	var err error
	if setting.DiscordChannel == "all" {
		blocks, err = channels.SlackBlocks()
		if err != nil {
			fmt.Printf("%v\n", errors.Wrapf(err, "Failed SlackBlocks"))
			return
		}
	} else {
		channel, ok := channels.Channels[setting.DiscordChannel]
		if !ok {
			fmt.Print("Failed to find channel")
		}
		blocks, err = channel.SlackBlock()
		if err != nil {
			fmt.Printf("%v\n", errors.Wrapf(err, "Failed SlackBlock"))
		}
	}
	var message = SlackHookBlock{
		Channel: setting.SlackChannel,
		Name:    "Discord Watcher",
		Blocks:  blocks,
	}
	message.Send()
}

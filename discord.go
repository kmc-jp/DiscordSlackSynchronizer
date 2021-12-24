package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	dp "github.com/kmc-jp/DiscordSlackSynchronizer/discord_plugin"
	"github.com/pkg/errors"
)

const DiscordAPIEndpoint = "https://discord.com/api"

type DiscordHandler struct {
	Session *discordgo.Session
	regExp  struct {
		UserID   *regexp.Regexp
		Channel  *regexp.Regexp
		ImageURI *regexp.Regexp

		replace *regexp.Regexp
		refURI  *regexp.Regexp
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
	d.regExp.replace = regexp.MustCompile(`\s*ss\/(.+)\/(.*)(\/)??\s*`)
	d.regExp.refURI = regexp.MustCompile(`\(RefURI:\s<https:.+>\)`)

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

	var reference *discordgo.Message
	var err error

	if m.Message != nil && m.Message.MessageReference != nil {
		reference, err = s.ChannelMessage(m.Message.MessageReference.ChannelID, m.Message.MessageReference.MessageID)
		if err != nil {
			log.Println(err)
			return
		}

		err = func() (err error) {
			if !d.regExp.replace.MatchString(m.Content) {
				return
			}

			id, err := d.parseUserName(reference.Author)
			if err != nil {
				return err
			}

			ids, err := dp.GetDiscordID(id)
			if err != nil {
				return err
			}

			var check bool
			for _, id := range ids {
				if id == m.Author.ID {
					check = true
				}
			}
			if !check {
				return fmt.Errorf("InvalidUpdateMessage")
			}

			err = d.deleteMessage(m.ChannelID, m.ID)
			if err != nil {
				log.Println(err)
			}

			var message DiscordMessage
			message.Message = reference
			message.Attachments = make([]DiscordAttachment, 0)

			for i := range message.Message.Attachments {
				if message.Message.Attachments[i] == nil {
					continue
				}

				var oldAtt = message.Message.Attachments[i]

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

			var newContent = reference.Content
			var newContentSlice = strings.Split(newContent, "\n")

			var refMatch = d.regExp.refURI.MatchString(newContentSlice[len(newContentSlice)-1])
			if refMatch {
				newContent = strings.Join(newContentSlice[1:len(newContentSlice)-1], "\n")
			}

			// replace and update message
			for _, pattern := range strings.Split(m.Content, "\n") {
				var matches = strings.Split(pattern, "/")
				var escapedMatch = []string{}
				var tmp string
				for i, match := range matches {
					if i < 1 {
						continue
					}
					if tmp != "" {
						match = tmp + "/" + match
						tmp = ""
					}
					if strings.HasSuffix(match, "\\") && !strings.HasSuffix(match, "\\\\") {
						tmp = strings.TrimSuffix(match, "\\")
						continue
					}

					escapedMatch = append(escapedMatch, match)
				}
				if len(escapedMatch) < 2 {
					return fmt.Errorf("Mal-formedExpression")
				}
				newContent = strings.ReplaceAll(newContent, escapedMatch[0], escapedMatch[1])
			}

			if refMatch {
				newContent = strings.Join(
					[]string{
						newContentSlice[0],
						newContent,
						newContentSlice[len(newContentSlice)-1],
					}, "\n",
				)
			}

			message.Content = newContent
			err = DiscordWebhook.Edit(message.ChannelID, message.ID, message, []DiscordFile{})
			if err != nil {
				log.Printf("EditError: %s\n", err)
				return
			}

			return
		}()
		if err == nil {
			return
		}
	}

	var name string = m.Member.Nick
	if name == "" {
		name = m.Author.Username
	}

	primaryID, err := dp.GetPrimaryID(m.Author.ID)
	if err != nil {
		log.Println(err)
		primaryID = m.Author.ID
	}

	var dMessage = DiscordMessage{
		AvaterURL: m.Author.AvatarURL(""),
		UserName:  fmt.Sprintf("%s(%s)", name, primaryID),
		Message: &discordgo.Message{
			ChannelID: m.ChannelID,
			Content:   m.Content,
		},
	}

	if reference != nil {
		var refText string
		var refSlice = strings.Split(reference.Content, "\n")
		if len(refSlice) > 1 {
			refText = refSlice[0] + "..."
		} else {
			refText = refSlice[0]
		}
		dMessage.Content = fmt.Sprintf("> %s\n%s\n(RefURI: <%s>)",
			refText,
			m.Content,
			fmt.Sprintf("https://discord.com/channels/%s/%s/%s",
				m.GuildID, reference.ChannelID, reference.ID,
			),
		)
	}

	// Delete message on Discord
	err = d.deleteMessage(m.ChannelID, m.ID)
	if err != nil {
		log.Println(err)
	} else {
		// if it was successed, send message by webhook
		DiscordWebhook.Send(m.ChannelID, m.ID, dMessage, []DiscordFile{})
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

	var content = dMessage.Content

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

	// Send message to Slack
	message.Send()
}

type VoiceEvent int

const (
	VoiceLeft         VoiceEvent = iota
	VoiceEmptied      VoiceEvent = iota
	VoiceStateChanged VoiceEvent = iota
	VoiceEntered      VoiceEvent = iota
)

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
		if len(channels.Channels[channel].Users) == 0 {
			d.sendVoiceState(setting, channels, VoiceEmptied)
		} else {
			d.sendVoiceState(setting, channels, VoiceLeft)
		}
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

		if !exists {
			d.sendVoiceState(setting, channels, VoiceEntered)
		} else if setting.Setting.SendMuteState {
			d.sendVoiceState(setting, channels, VoiceStateChanged)
		}
	}
}

func (d *DiscordHandler) sendVoiceState(setting ChannelSetting, channels *VoiceChannels, event VoiceEvent) {
	if setting.SlackChannel == "" {
		return
	}
	if !setting.Setting.SendVoiceState {
		return
	}
	var blocks []json.RawMessage
	var err error
	if setting.DiscordChannel == "all" {
		blocks, err = channels.SlackBlocksMultiChannel()
		if err != nil {
			fmt.Printf("%v\n", errors.Wrapf(err, "Failed SlackBlocks"))
			return
		}
	} else {
		channel, ok := channels.Channels[setting.DiscordChannel]
		if !ok {
			fmt.Print("Failed to find channel")
		}
		blocks, err = channel.SlackBlocksSingleChannel()
		if err != nil {
			fmt.Printf("%v\n", errors.Wrapf(err, "Failed SlackBlock"))
		}
	}
	var message = SlackHookBlock{
		Channel:   setting.SlackChannel,
		Name:      "Discord Watcher",
		Blocks:    blocks,
		IconEmoji: "discord",
	}
	switch event {
	case VoiceEntered:
		slackIndicator.Popup(message)
	case VoiceLeft:
		slackIndicator.Update(message)
	case VoiceStateChanged:
		slackIndicator.Update(message)
	case VoiceEmptied:
		slackIndicator.Remove(message.Channel)
	}
}

func (d *DiscordHandler) deleteMessage(channelID, messageID string) (err error) {
	req, err := http.NewRequest(
		"DELETE",
		fmt.Sprintf("%s/channels/%s/messages/%s",
			DiscordAPIEndpoint, channelID, messageID,
		),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", d.Session.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		return fmt.Errorf("FailedToDeleteMessage")
	}

	return nil
}

func (d *DiscordHandler) parseUserName(m *discordgo.User) (string, error) {
	var nameSlice = strings.Split(m.Username, "(")
	if len(nameSlice) < 1 {
		return "", fmt.Errorf("Mal-FormedName")
	}

	var id = strings.Split(nameSlice[len(nameSlice)-1], ")")[0]
	return id, nil
}

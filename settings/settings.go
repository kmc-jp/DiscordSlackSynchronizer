package settings

import (
	"encoding/json"
	"io/ioutil"
	"log"

	"github.com/pkg/errors"
)

type Handler struct {
	channelMap       *ChannelMap
	settingsFilePath string
}

//SlackDiscordTable dict of Channel
type SlackDiscordTable struct {
	Discord       string           `json:"discord_server"`
	Channel       []ChannelSetting `json:"channel"`
	SlackSuffix   string           `json:"slack_suffix"`
	DiscordSuffix string           `json:"discord_suffix"`
}

//ChannelSetting Put send settings
type ChannelSetting struct {
	Comment        string      `json:"comment"`
	SlackChannel   string      `json:"slack"`
	DiscordChannel string      `json:"discord"`
	Setting        SendSetting `json:"setting"`
	Webhook        string      `json:"hook"`
}

//SendSetting put send setting
type SendSetting struct {
	SlackToDiscord           bool `json:"slack2discord"`
	DiscordToSlack           bool `json:"discord2slack"`
	ShowChannelName          bool `json:"ShowChannelName"`
	SendVoiceState           bool `json:"SendVoiceState"`
	SendMuteState            bool `json:"SendMuteState"`
	CreateSlackChannelOnSend bool `json:"CreateSlackChannelOnSend"`
	MuteSlackUsers           Users
}

func New(slackToken, discordToken, settingsFilePath string) *Handler {
	return &Handler{
		settingsFilePath: settingsFilePath,
		channelMap:       NewChannelMap(slackToken, discordToken),
	}
}

func (s Handler) GetChannelMap() ([]SlackDiscordTable, error) {
	var dict []SlackDiscordTable

	dataBytes, err := ioutil.ReadFile(s.settingsFilePath)
	if err != nil {
		return nil, errors.Wrap(err, "ReadFile")
	}

	err = json.Unmarshal(dataBytes, &dict)
	if err != nil {
		return nil, errors.Wrap(err, "Unmarshal")
	}

	return dict, nil
}

func (s Handler) WriteChannelMap(dict []SlackDiscordTable) error {
	b, err := json.MarshalIndent(dict, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(s.settingsFilePath, b, 0644)
}

func (s Handler) FindSlackChannel(DiscordChannel string, guildID string) ChannelSetting {
	dict, err := s.GetChannelMap()
	if err != nil {
		log.Println(errors.Wrap(err, "GetChannelMap"))
	}

	var result ChannelSetting
	if dict == nil {
		return ChannelSetting{}
	}
	for _, c := range dict {
		s.channelMap.UpdateChannels(c.Discord, c.SlackSuffix, c.DiscordSuffix)

		if c.Discord == guildID {
			for _, channelSet := range c.Channel {
				if channelSet.DiscordChannel == DiscordChannel {
					result = channelSet
					return result
				}
				// Complete Transfer
				if channelSet.SlackChannel == "all" && channelSet.DiscordChannel == "all" {
					result = channelSet
					result.SlackChannel = s.channelMap.DiscordToSlack(
						DiscordChannel, result.Setting.CreateSlackChannelOnSend)
					if result.SlackChannel == "" {
						continue
					}
					result.DiscordChannel = DiscordChannel
					return result
				}
				// All-In-One Transfer
				if channelSet.DiscordChannel == "all" {
					result = channelSet
					return result
				}
			}
		}
	}
	return result
}

// FindDiscordChannel find Discord channel from slack channel id
func (s Handler) FindDiscordChannel(SlackChannel string) (ChannelSetting, string) {
	dict, err := s.GetChannelMap()
	if err != nil {
		log.Println(errors.Wrap(err, "GetChannelMap"))
	}

	if dict == nil {
		return ChannelSetting{}, ""
	}

	for _, c := range dict {
		s.channelMap.UpdateChannels(c.Discord, c.SlackSuffix, c.DiscordSuffix)

		for _, channelSet := range c.Channel {
			if channelSet.SlackChannel == SlackChannel && channelSet.DiscordChannel != "all" {
				return channelSet, c.Discord
			}
			// Complete Transfer
			if channelSet.SlackChannel == "all" && channelSet.DiscordChannel == "all" {
				result := channelSet
				result.DiscordChannel = s.channelMap.SlackToDiscord(SlackChannel)
				if result.DiscordChannel == "" {
					continue
				}
				result.SlackChannel = SlackChannel
				return result, result.DiscordChannel
			}
		}
	}
	return ChannelSetting{}, ""
}

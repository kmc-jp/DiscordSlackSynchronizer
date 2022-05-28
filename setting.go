package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

type SettingsHandler struct {
	channelMap *ChannelMap
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
	SlackChannel   string `json:"slack"`
	DiscordChannel string `json:"discord"`
	Setting        SendSetting
	Webhook        string `json:"hook"`
}

//SendSetting put send setting
type SendSetting struct {
	SlackToDiscord           bool 	`json:"slack2discord"`
	DiscordToSlack           bool 	`json:"discord2slack"`
	ShowChannelName          bool 	`json:"ShowChannelName"`
	SendVoiceState           bool 	`json:"SendVoiceState"`
	SendMuteState            bool 	`json:"SendMuteState"`
	CreateSlackChannelOnSend bool   `json:"CreateSlackChannelOnSend"`
	MuteSlackUserID				 []string `json:"MuteSlackUserID"`
}

func NewSettingsHandler(slackToken, discordToken string) *SettingsHandler {
	return &SettingsHandler{
		channelMap: NewChannelMap(slackToken, discordToken),
	}
}

func (s SettingsHandler) readChannelMap() []SlackDiscordTable {
	var dict []SlackDiscordTable
	dataBytes, err := ioutil.ReadFile(SettingsFile)
	if err != nil {
		fmt.Printf("%v", err.Error())
		panic("invalid permission: slackMap.json")

	}

	err = json.Unmarshal(dataBytes, &dict)
	if err != nil {
		fmt.Printf("%v", err.Error())
		panic("invalid syntax: slackMap.json")

	}

	return dict
}

func (s SettingsHandler) FindSlackChannel(DiscordChannel string, guildID string) ChannelSetting {
	var dict = s.readChannelMap()

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
func (s SettingsHandler) FindDiscordChannel(SlackChannel string) (ChannelSetting, string) {
	var dict = s.readChannelMap()
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

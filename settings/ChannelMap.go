package settings

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/slack-go/slack"
)

type ChannelMap struct {
	slack   *slack.Client
	discord *discordgo.Session

	slackToDiscord   map[string]string
	discordToSlack   map[string]string
	slackIDByName    map[string]string
	slackNameByID    map[string]string
	discordIDBylName map[string]string
	discordNameByID  map[string]string
	slackSuffix      string
	discordSuffix    string
	lastUpdated      time.Time
	mu               sync.RWMutex
}

const ChannelMapUpdateIntervals time.Duration = 20 * time.Second

func NewChannelMap(slackToken, discordToken string) *ChannelMap {
	discord, _ := discordgo.New("Bot " + discordToken)
	return &ChannelMap{
		slack:   slack.New(slackToken),
		discord: discord,

		slackToDiscord:   map[string]string{},
		discordToSlack:   map[string]string{},
		slackIDByName:    map[string]string{},
		slackNameByID:    map[string]string{},
		discordIDBylName: map[string]string{},
		discordNameByID:  map[string]string{},
	}
}

func (c *ChannelMap) SlackToDiscord(slackID string) string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.slackToDiscord[slackID]
}
func (c *ChannelMap) DiscordToSlack(discordID string, createIfNotExist bool) string {
	channel := func() string {
		c.mu.RLock()
		defer c.mu.RUnlock()
		return c.discordToSlack[discordID]
	}()
	if channel != "" {
		return channel
	}
	if createIfNotExist {
		name := c.discordNameByID[discordID]
		name = fmt.Sprintf("%s%s", strings.TrimSuffix(name, c.discordSuffix), c.slackSuffix)
		channel = c.CreateChannel(name)
		return channel
	}
	return ""
}
func (c *ChannelMap) CreateChannel(name string) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	channel, err := c.slack.CreateConversation(name, false)
	if err != nil {
		fmt.Printf("Error creating conversation: %v\n", err)
		return ""
	}
	c.slackIDByName[channel.ID] = channel.Name
	c.generateMap()
	return channel.ID
}

func (c *ChannelMap) generateMap() {
	for slackName, slackID := range c.slackIDByName {
		if strings.HasSuffix(slackName, c.slackSuffix) {
			slackNamePrefix := strings.TrimSuffix(slackName, c.slackSuffix)
			discordName := fmt.Sprintf("%s%s", slackNamePrefix, c.discordSuffix)
			discordID, ok := c.discordIDBylName[discordName]
			if ok {
				c.slackToDiscord[slackID] = discordID
				c.discordToSlack[discordID] = slackID
			}
		}
	}
}

func (c *ChannelMap) FetchSlackChannels() {
	cursor := ""
	for {
		var err error
		var channels []slack.Channel
		channels, cursor, err = c.slack.GetConversations(&slack.GetConversationsParameters{
			Cursor: cursor,
			Limit:  1000,
		})
		if err != nil {
			fmt.Printf("Error fetchSlackChannels: %v", err)
			return
		}
		for _, channel := range channels {
			c.slackIDByName[channel.Name] = channel.ID
			c.slackIDByName[channel.ID] = channel.Name
		}
		if cursor == "" {
			break
		}
	}
}

func (c *ChannelMap) FetchDiscordChannel(guildID string) {
	channels, _ := c.discord.GuildChannels(guildID)

	for _, channel := range channels {
		if channel.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		c.discordIDBylName[channel.Name] = channel.ID
		c.discordNameByID[channel.ID] = channel.Name
	}
}

func (c *ChannelMap) UpdateChannels(guildID string, slackSuffix string, discordSuffix string) {
	c.slackSuffix = slackSuffix
	c.discordSuffix = discordSuffix
	now := time.Now()
	if now.Sub(c.lastUpdated) < ChannelMapUpdateIntervals {
		return
	}
	c.lastUpdated = now

	defer func() {
		c.FetchSlackChannels()
		c.FetchDiscordChannel(guildID)

		c.mu.Lock()
		defer c.mu.Unlock()
		c.generateMap()
	}()
}

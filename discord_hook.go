package main

import (
	"fmt"
	"sync"

	"github.com/bwmarrin/discordgo"
)

//DiscordHookMessage DiscordにIncommingWebhook経由のMessage送信形式
type DiscordHookMessage struct {
	Channel string `json:"channel_id"`
	Server  string `json:"guild_id"`
	Name    string `json:"username"`
	Text    string `json:"content"`
	IconURL string `json:"avatar_url"`
}

type DiscordWebhookType struct {
	webhookByChannelID map[string]*discordgo.Webhook
	createWebhookLock  map[string]*sync.RWMutex
}

var DiscordWebhook DiscordWebhookType = DiscordWebhookType{
	webhookByChannelID: map[string]*discordgo.Webhook{},
	createWebhookLock:  map[string]*sync.RWMutex{},
}

func (d *DiscordWebhookType) createWebhook(channelID string) *discordgo.Webhook {
	d.createWebhookLock[channelID].Lock()
	defer d.createWebhookLock[channelID].Unlock()
	webhooks, err := Discord.Session.ChannelWebhooks(channelID)
	if err != nil {
		fmt.Printf("Error getting webhook: %v\n", err)
		return nil
	}
	if len(webhooks) == 0 {
		webhook, err := Discord.Session.WebhookCreate(channelID, "Slack Synchronizer", "a")
		if err != nil {
			fmt.Printf("Error creating webhook: %v\n", err)
			return nil
		}
		return webhook
	}
	return webhooks[0]
}

func (d *DiscordWebhookType) Get(channelID string) *discordgo.Webhook {
	_, ok := d.createWebhookLock[channelID]
	if !ok {
		d.createWebhookLock[channelID] = &sync.RWMutex{}
	}
	webhook := func() *discordgo.Webhook {
		d.createWebhookLock[channelID].RLock()
		defer d.createWebhookLock[channelID].RUnlock()
		return d.webhookByChannelID[channelID]
	}()
	if webhook != nil {
		return webhook
	}
	webhook = d.createWebhook(channelID)
	d.webhookByChannelID[channelID] = webhook
	return webhook
}

//Send json形式で指定したURLにPOSTする。
func (h *DiscordHookMessage) Send() {
	if h.Channel == "" {
		fmt.Printf("Faild discord send because h.Channel is empty: %+v\n", h)
		return
	}
	webhook := DiscordWebhook.Get(h.Channel)
	if webhook == nil {
		fmt.Printf("Faild get webhook\n")
		return
	}

	data := &discordgo.WebhookParams{
		Content:   h.Text,
		Username:  h.Name,
		AvatarURL: h.IconURL,
	}
	_, err := Discord.Session.WebhookExecute(webhook.ID, webhook.Token, false, data)
	if err != nil {
		fmt.Printf("MessageSendError(Discord):\n   %s\n", err.Error())
		return
	}
}

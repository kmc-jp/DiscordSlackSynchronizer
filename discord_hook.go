package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"sync"

	"github.com/bwmarrin/discordgo"
)

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

func (d *DiscordWebhookType) send(method, channelID, messageID string, message DiscordMessage, files []DiscordFile) (err error) {
	var hook = d.Get(channelID)

	if err != nil {
		return
	}

	var body = new(bytes.Buffer)

	var mw = multipart.NewWriter(body)

	var mh = make(textproto.MIMEHeader)
	mh.Set("Content-Type", "application/json")
	mh.Set(`Content-Disposition`, `form-data; name="payload_json"`)

	pw, err := mw.CreatePart(mh)
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(message, "", "    ")

	var jsonBuf = bytes.NewBuffer(b)
	io.Copy(pw, jsonBuf)

	for i, file := range files {
		var mh = make(textproto.MIMEHeader)
		mh.Set("Content-Type", file.ContentType)
		mh.Set(`Content-Disposition`, fmt.Sprintf(`form-data; name="files[%d]"; filename="%s"`, i, file.FileName))

		pw, err := mw.CreatePart(mh)
		if err != nil {
			return err
		}

		io.Copy(pw, file.Reader)
	}

	mw.Close()

	var req *http.Request
	switch method {
	case "EDIT":
		req, err = http.NewRequest(
			"PATCH",
			fmt.Sprintf("%s/webhooks/%s/%s/messages/%s",
				DiscordAPIEndpoint, hook.ID, hook.Token, messageID,
			),
			body,
		)
	case "SEND":
		req, err = http.NewRequest(
			"POST",
			fmt.Sprintf("%s/webhooks/%s/%s",
				DiscordAPIEndpoint, hook.ID, hook.Token,
			),
			body,
		)
	}

	if err != nil {
		return
	}

	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return
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

func (d *DiscordWebhookType) Edit(channelID, messageID string, message DiscordMessage, files []DiscordFile) (err error) {
	return DiscordWebhook.send("EDIT", channelID, messageID, message, files)
}

func (d *DiscordWebhookType) Send(channelID, messageID string, message DiscordMessage, files []DiscordFile) (err error) {
	return DiscordWebhook.send("SEND", channelID, messageID, message, files)
}

package discord_webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"sync"

	"github.com/bwmarrin/discordgo"
)

const DiscordAPIEndpoint = "https://discord.com/api"

type Handler struct {
	webhookByChannelID map[string]*discordgo.Webhook
	createWebhookLock  map[string]*sync.RWMutex
	token              string
}

type File struct {
	FileName    string
	Reader      io.Reader
	ContentType string
}

type Message struct {
	*discordgo.Message
	Components  []Component  `json:"components,omitempty"`
	AvaterURL   string       `json:"avatar_url,omitempty"`
	Attachments []Attachment `json:"attachments"`
	UserName    string       `json:"username,omitempty"`
}

type Component struct {
	Type        int              `json:"type"`
	CustomID    string           `json:"custom_id,omitempty"`
	Disabled    bool             `json:"disabled,omitempty"`
	Style       int              `json:"style,omitempty"`
	Label       string           `json:"label,omitempty"`
	URL         string           `json:"url,omitempty"`
	Placeholder string           `json:"placeholder,omitempty"`
	MinValues   int              `json:"min_values,omitempty"`
	MaxValues   int              `json:"max_values,omitempty"`
	Components  []Component      `json:"components,omitempty"`
	Emoji       *discordgo.Emoji `json:"emoji,omitempty"`
	Options     []Option         `json:"options,omitempty"`
}

type Option struct {
	Label       string           `json:"label"`
	Value       string           `json:"value"`
	Description string           `json:"description,omitempty"`
	Emoji       *discordgo.Emoji `json:"emoji,omitempty"`
	Default     bool             `json:"default,omitempty"`
}

type Attachment struct {
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	ID          int    `json:"id"`
	ProxyURL    string `json:"proxy_url,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Size        int    `json:"size,omitempty"`
}

func New(token string) *Handler {
	return &Handler{
		webhookByChannelID: map[string]*discordgo.Webhook{},
		createWebhookLock:  map[string]*sync.RWMutex{},
		token:              token,
	}
}

func (h *Handler) Reset() {
	h = &Handler{
		webhookByChannelID: map[string]*discordgo.Webhook{},
		createWebhookLock:  map[string]*sync.RWMutex{},
	}
}

func (h *Handler) createWebhook(channelID string) *discordgo.Webhook {
	h.createWebhookLock[channelID].Lock()
	defer h.createWebhookLock[channelID].Unlock()
	webhooks, err := h.getChannelWebhook(channelID)
	if err != nil {
		fmt.Printf("Error getting webhook: %v\n", err)
		return nil
	}
	if len(webhooks) == 0 {
		webhook, err := h.createChannelWebhook(channelID, "Slack Synchronizer", "a")
		if err != nil {
			fmt.Printf("Error creating webhook: %v\n", err)
			return nil
		}
		return webhook
	}
	return webhooks[0]
}

func (h *Handler) getChannelWebhook(channelID string) ([]*discordgo.Webhook, error) {
	var body = new(bytes.Buffer)

	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/channels/%s/webhooks",
			DiscordAPIEndpoint, channelID,
		),
		body,
	)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+h.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var webhook []*discordgo.Webhook

	err = json.NewDecoder(resp.Body).Decode(&webhook)

	return webhook, err
}

func (h *Handler) createChannelWebhook(method, channelID, name string) (*discordgo.Webhook, error) {
	var body = new(bytes.Buffer)
	var req *http.Request
	var err error

	type CreateWebhookOption struct {
		Name string `json:"name"`
	}

	var webhookOpt = CreateWebhookOption{
		Name: name,
	}
	err = json.NewEncoder(body).Encode(webhookOpt)
	if err != nil {
		return nil, err
	}

	req, err = http.NewRequest(
		"POST",
		fmt.Sprintf("%s/channels/%s/webhooks",
			DiscordAPIEndpoint, channelID,
		),
		body,
	)

	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bot "+h.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var webhook discordgo.Webhook

	err = json.NewDecoder(resp.Body).Decode(&webhook)

	return &webhook, err
}

func (h *Handler) send(method, channelID, messageID string, message Message, files []File) (err error) {
	var hook = h.Get(channelID)

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

func (h *Handler) Get(channelID string) *discordgo.Webhook {
	_, ok := h.createWebhookLock[channelID]
	if !ok {
		h.createWebhookLock[channelID] = &sync.RWMutex{}
	}
	webhook := func() *discordgo.Webhook {
		h.createWebhookLock[channelID].RLock()
		defer h.createWebhookLock[channelID].RUnlock()
		return h.webhookByChannelID[channelID]
	}()
	if webhook != nil {
		return webhook
	}
	webhook = h.createWebhook(channelID)
	h.webhookByChannelID[channelID] = webhook
	return webhook
}

func (h *Handler) Edit(channelID, messageID string, message Message, files []File) (err error) {
	return h.send("EDIT", channelID, messageID, message, files)
}

func (h *Handler) Send(channelID, messageID string, message Message, files []File) (err error) {
	return h.send("SEND", channelID, messageID, message, files)
}

func (h *Handler) GetMessage(channelID, messageID string) (messages discordgo.Message, err error) {
	var requestAttr = make(url.Values)

	var client = http.DefaultClient
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/channels/%s/messages/%s?%s", DiscordAPIEndpoint, channelID, messageID, requestAttr.Encode()),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+h.token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var responseAttr discordgo.Message
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

func (h *Handler) GetMessages(channelID string, around string) (messages []discordgo.Message, err error) {
	var requestAttr = make(url.Values)

	requestAttr.Set("limit", "100")
	if around != "" {
		requestAttr.Set("around", around)
	}

	var client = http.DefaultClient
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/channels/%s/messages?%s", DiscordAPIEndpoint, channelID, requestAttr.Encode()),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+h.token)

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
				buf := new(bytes.Buffer)
				io.Copy(buf, resp.Body)
				fmt.Println(buf)
			}
		}()
		return json.NewDecoder(resp.Body).Decode(&responseAttr)
	}()
	if err != nil {
		return
	}

	return responseAttr, nil
}

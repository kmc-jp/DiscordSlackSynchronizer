package discord_webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/pkg/errors"
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

// A Channel holds all data related to an individual Discord channel.
type Channel struct {
	ID                   string                           `json:"id"`
	GuildID              string                           `json:"guild_id"`
	Name                 string                           `json:"name"`
	Topic                string                           `json:"topic"`
	Type                 discordgo.ChannelType            `json:"type"`
	LastMessageID        string                           `json:"last_message_id"`
	LastPinTimestamp     time.Time                        `json:"last_pin_timestamp"`
	NSFW                 bool                             `json:"nsfw"`
	Icon                 string                           `json:"icon"`
	Position             int                              `json:"position"`
	Bitrate              int                              `json:"bitrate"`
	Recipients           []*discordgo.User                `json:"recipients"`
	Messages             []*Message                       `json:"-"`
	PermissionOverwrites []*discordgo.PermissionOverwrite `json:"permission_overwrites"`
	UserLimit            int                              `json:"user_limit"`
	ParentID             string                           `json:"parent_id"`
	RateLimitPerUser     int                              `json:"rate_limit_per_user"`
	OwnerID              string                           `json:"owner_id"`
	ApplicationID        string                           `json:"application_id"`
}

type Message struct {
	Components  []Component  `json:"components,omitempty"`
	AvaterURL   string       `json:"avatar_url,omitempty"`
	Attachments []Attachment `json:"attachments"`
	UserName    string       `json:"username,omitempty"`

	ID               string                        `json:"id"`
	ChannelID        string                        `json:"channel_id"`
	GuildID          string                        `json:"guild_id,omitempty"`
	Content          string                        `json:"content"`
	Timestamp        time.Time                     `json:"timestamp"`
	EditedTimestamp  *time.Time                    `json:"edited_timestamp"`
	MentionRoles     []string                      `json:"mention_roles"`
	TTS              bool                          `json:"tts"`
	MentionEveryone  bool                          `json:"mention_everyone"`
	Author           *discordgo.User               `json:"author"`
	Embeds           []*discordgo.MessageEmbed     `json:"embeds"`
	Mentions         []*discordgo.User             `json:"mentions"`
	Reactions        []*discordgo.MessageReactions `json:"reactions"`
	Pinned           bool                          `json:"pinned"`
	Type             discordgo.MessageType         `json:"type"`
	WebhookID        string                        `json:"webhook_id"`
	Member           *discordgo.Member             `json:"member"`
	MentionChannels  []*discordgo.Channel          `json:"mention_channels"`
	Activity         *discordgo.MessageActivity    `json:"activity"`
	Application      *discordgo.MessageApplication `json:"application"`
	MessageReference *discordgo.MessageReference   `json:"message_reference"`
	Flags            discordgo.MessageFlags        `json:"flags"`
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
	ID          string `json:"id"`
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
		webhook, err := h.createChannelWebhook(channelID, "Slack Synchronizer")
		if err != nil {
			fmt.Printf("Error: CreateChannelWebhook: %v\n", err)
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

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadAll")
	}

	var webhook []*discordgo.Webhook
	err = json.Unmarshal(buf, &webhook)

	return webhook, errors.Wrap(err, "Unmarshal")
}

func (h *Handler) createChannelWebhook(channelID, name string) (*discordgo.Webhook, error) {
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
		return nil, errors.Wrap(err, "Encode")
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
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var webhook discordgo.Webhook

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadAll")
	}

	err = json.Unmarshal(buf, &webhook)
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal: %s", buf)
	}
	if webhook.ID == "" {
		return nil, errors.New(fmt.Sprintf("API: %s", buf))
	}

	return &webhook, nil
}

func (h *Handler) send(method, channelID, messageID string, message Message, wait bool, files []File) (newMessage *Message, err error) {
	var hook = h.Get(channelID)
	if files == nil {
		files = []File{}
	}

	var body = new(bytes.Buffer)

	var mw = multipart.NewWriter(body)

	var mh = make(textproto.MIMEHeader)
	mh.Set("Content-Type", "application/json")
	mh.Set(`Content-Disposition`, `form-data; name="payload_json"`)

	pw, err := mw.CreatePart(mh)
	if err != nil {
		return nil, errors.Wrap(err, "CreatingPart")
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
			return nil, errors.Wrap(err, "CreatingPartAtFor")
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
		var reqURI = fmt.Sprintf("%s/webhooks/%s/%s",
			DiscordAPIEndpoint, hook.ID, hook.Token,
		)
		if wait {
			reqURI += "?wait=true"
		}
		req, err = http.NewRequest(
			"POST",
			reqURI,
			body,
		)
	}

	if err != nil {
		return
	}

	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Sending")
	}
	defer resp.Body.Close()

	var responseAttr Message

	buf, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		errors.Wrap(err, "ReadResponseBody")
		return
	}

	if len(buf) < 1 {
		return &responseAttr, nil
	}

	err = json.Unmarshal(buf, &responseAttr)

	return &responseAttr, errors.Wrapf(err, "JsonParsing: %s", buf)
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

func (h *Handler) Edit(channelID, messageID string, message Message, files []File) (*Message, error) {
	return h.send("EDIT", channelID, messageID, message, false, files)
}

func (h *Handler) Send(channelID string, message Message, wait bool, files []File) (*Message, error) {
	return h.send("SEND", channelID, "", message, wait, files)
}

func (h *Handler) GetGuildChannels(guildID string) (channels []discordgo.Channel, err error) {
	var client = http.DefaultClient
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/guilds/%s/channels", DiscordAPIEndpoint, guildID),
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

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadAll")
	}

	err = json.Unmarshal(body, &channels)
	if err != nil {
		return nil, fmt.Errorf("UnmarshalJSON: %s body: %s", err.Error(), body)
	}

	return
}

func (h *Handler) GetMessage(channelID, messageID string) (message discordgo.Message, err error) {
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

	body, err := ioutil.ReadAll(resp.Body)

	var responseAttr []discordgo.Message

	err = json.Unmarshal(body, &responseAttr)
	if err != nil {
		err = errors.Wrapf(err, "JsonUnmarshal: %s", body)
		return
	}

	return responseAttr, nil
}

func (h *Handler) GetChannel(channelID string) (*Channel, error) {
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/channels/%s", DiscordAPIEndpoint, channelID),
		nil,
	)
	if err != nil {
		return nil, errors.Wrap(err, "NewRequest")
	}

	req.Header.Set("Authorization", "Bot "+h.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "DoRequest")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadAll")
	}

	var channel Channel
	err = json.Unmarshal(body, &channel)

	return &channel, errors.Wrap(err, "Unmarshal")
}

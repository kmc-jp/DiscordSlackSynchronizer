package slack_webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

const SlackAPIEndpoint = "https://slack.com/api"

type Handler struct {
	token string
}

//HookMessage SlackにIncommingWebhook経由のMessage送信形式
type Message struct {
	TS          string       `json:"ts,omitempty"`
	Channel     string       `json:"channel"`
	Text        string       `json:"text,omitempty"`
	Blocks      []BlockBase  `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`

	LinkNames bool   `json:"link_names,omitempty"`
	Username  string `json:"username,omitempty"`
	AsUser    bool   `json:"as_user,omitempty"`

	Parse           string `json:"parse,omitempty"`
	ThreadTimestamp string `json:"thread_ts,omitempty"`
	ReplyBroadcast  bool   `json:"reply_broadcast,omitempty"`

	UnfurlLinks bool `json:"unfurl_links"`
	UnfurlMedia bool `json:"unfurl_media"`

	IconURL   string `json:"icon_url,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
}

// Attachment contains all the information for an attachment
type Attachment struct {
	Color    string `json:"color,omitempty"`
	Fallback string `json:"fallback,omitempty"`

	CallbackID string `json:"callback_id,omitempty"`
	ID         int    `json:"id,omitempty"`

	AuthorID      string `json:"author_id,omitempty"`
	AuthorName    string `json:"author_name,omitempty"`
	AuthorSubname string `json:"author_subname,omitempty"`
	AuthorLink    string `json:"author_link,omitempty"`
	AuthorIcon    string `json:"author_icon,omitempty"`

	Title     string `json:"title,omitempty"`
	TitleLink string `json:"title_link,omitempty"`
	Pretext   string `json:"pretext,omitempty"`
	Text      string `json:"text,omitempty"`

	ImageURL string `json:"image_url,omitempty"`
	ThumbURL string `json:"thumb_url,omitempty"`

	ServiceName string `json:"service_name,omitempty"`
	ServiceIcon string `json:"service_icon,omitempty"`
	FromURL     string `json:"from_url,omitempty"`
	OriginalURL string `json:"original_url,omitempty"`

	Fields     []slack.AttachmentField  `json:"fields,omitempty"`
	Actions    []slack.AttachmentAction `json:"actions,omitempty"`
	MarkdownIn []string                 `json:"mrkdwn_in,omitempty"`

	Footer     string `json:"footer,omitempty"`
	FooterIcon string `json:"footer_icon,omitempty"`

	Ts json.Number `json:"ts,omitempty"`
}

type GetConversationHistoryParameters struct {
	ChannelID string `json:"channel"`
	Cursor    string `json:"cursor,omitempty"`
	Inclusive bool   `json:"inclusive,omitempty"`
	Latest    string `json:"latest,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Oldest    string `json:"oldest,omitempty"`
}

func New(token string) *Handler {
	return &Handler{token}
}

func (s *Handler) send(jsondataBytes []byte, method string) (string, error) {
	var req *http.Request
	switch method {
	case "update":
		req, _ = http.NewRequest("POST", SlackAPIEndpoint+"/chat.update", bytes.NewBuffer(jsondataBytes))
	case "delete":
		req, _ = http.NewRequest("POST", SlackAPIEndpoint+"/chat.delete", bytes.NewBuffer(jsondataBytes))
	case "send":
		req, _ = http.NewRequest("POST", SlackAPIEndpoint+"/chat.postMessage", bytes.NewBuffer(jsondataBytes))
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("MessageSendError(Slack): %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("readall: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed send slack: body: %s", body)
	}
	var r struct {
		OK bool   `json:"ok"`
		TS string `json:"ts"`
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}
	if !r.OK {
		return "", fmt.Errorf("failed send slack: body: %s", body)
	}

	return r.TS, nil
}

//Send json形式で指定したURLにPOSTする。
func (s *Handler) Send(message Message) (string, error) {
	jsonDataBytes, _ := json.Marshal(message)
	return s.send(jsonDataBytes, "send")
}

func (s *Handler) Update(message Message) (string, error) {
	jsonDataBytes, _ := json.Marshal(message)
	return s.send(jsonDataBytes, "update")
}

func (s Handler) Remove(channel, ts string) (string, error) {
	type RemoveMessage struct {
		Channel string `json:"channel"`
		TS      string `json:"ts"`
	}
	var removeMessage = &RemoveMessage{
		Channel: channel,
		TS:      ts,
	}
	jsonDataBytes, _ := json.Marshal(removeMessage)
	return s.send(jsonDataBytes, "delete")
}

func (s *Handler) GetMessages(channelID, timestamp string, limit int) ([]Message, error) {
	var requestAttr = url.Values{}

	requestAttr.Set("channel", channelID)
	requestAttr.Set("latest", timestamp)
	requestAttr.Set("limit", strconv.Itoa(limit))
	requestAttr.Set("inclusive", "true")

	msgs, err := s.getMessages(requestAttr.Encode())
	if err != nil {
		return nil, errors.Wrap(err, "getMessages")
	}

	return msgs, nil
}

func (s *Handler) GetMessage(channelID, timestamp string) (string, error) {
	msgs, err := s.GetMessages(channelID, timestamp, 1)
	if len(msgs) < 1 {
		return "", errors.Wrap(err, "NotFound")
	}
	return msgs[0].Text, err
}

func (s *Handler) getMessages(query string) ([]Message, error) {
	req, _ := http.NewRequest("GET", SlackAPIEndpoint+"/conversations.history?"+query, nil)

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("MessageSendError(Slack): %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("readall: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed send slack: body: %s", body)
	}

	var r struct {
		OK       bool      `json:"ok"`
		Messages []Message `json:"messages"`
	}

	err = json.Unmarshal(body, &r)
	if err != nil {
		return nil, fmt.Errorf("unmarshal: %w", err)
	}
	if !r.OK {
		return nil, fmt.Errorf("failed send slack: body: %s", body)
	}

	if err != nil {
		return nil, err
	}

	return r.Messages, nil
}

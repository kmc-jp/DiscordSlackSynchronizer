package slack_webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/slack-go/slack"
)

const SlackAPIEndpoint = "https://slack.com/api"

type Handler struct {
	token string
}

//HookMessage SlackにIncommingWebhook経由のMessage送信形式
type Message struct {
	TS          string             `json:"ts,omitempty"`
	Channel     string             `json:"channel"`
	Text        string             `json:"text,omitempty"`
	Blocks      []BlockBase        `json:"blocks,omitempty"`
	Attachments []slack.Attachment `json:"attachments,omitempty"`

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

type BlockBase struct {
	Type     string         `json:"type"`
	Elements []BlockElement `json:"elements,omitempty"`
}

type BlockElement struct {
	Type     string `json:"type"`
	ImageURL string `json:"image_url,omitempty"`
	AltText  string `json:"alt_text,omitempty"`
	Text     string `json:"text,omitempty"`
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

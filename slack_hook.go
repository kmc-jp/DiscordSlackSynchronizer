package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pkg/errors"
)

//SlackHookMessage SlackにIncommingWebhook経由のMessage送信形式
type SlackHookMessage struct {
	AsUser      bool              `json:"as_user,omitempty"`
	Channel     string            `json:"channel"`
	Name        string            `json:"username"`
	Text        string            `json:"text"`
	IconURL     string            `json:"icon_url"`
	Attachments []SlackAttachment `json:"attachments"`
}

type SlackHookBlock struct {
	AsUser    bool              `json:"as_user,omitempty"`
	Channel   string            `json:"channel"`
	Name      string            `json:"username"`
	IconEmoji string            `json:"icon_emoji"`
	Text      string            `json:"text"`
	UnfLinks  bool              `json:"unfurl_links"`
	Blocks    []json.RawMessage `json:"blocks"`
}

type SlackAttachment struct {
	AuthorIcon string `json:"author_icon,omitempty"`
	AuthorLink string `json:"author_link,omitempty"`
	AuthorName string `json:"author_name,omitempty"`

	FallBack string `json:"fallback,omitempty"`

	Footer     string `json:"footer,omitempty"`
	FooterIcon string `json:"footer_icon,omitempty"`

	ImageURL string `json:"image_url,omitempty"`

	MrkdownIn string `json:"mrkdwn_in,omitempty"`
	Pretext   string `json:"pretext,omitempty"`
	Text      string `json:"text,omitempty"`

	ThumbURL  string `json:"thumb_url,omitempty"`
	Title     string `json:"title,omitempty"`
	TitleLink string `json:"title_link,omitempty"`
	TS        string `json:"ts,omitempty"`

	Fields SlackFields `json:"fields,omitempty"`
}

type SlackFields struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
	Short bool   `json:"short,omitempty"`
}

const SlackMessagePostURI = "https://slack.com/api/chat.postMessage"

//Send json形式で指定したURLにPOSTする。
func (h *SlackHookMessage) Send() (err error) {
	jsondataBytes, _ := json.Marshal(h)

	req, _ := http.NewRequest("POST", SlackMessagePostURI, bytes.NewBuffer(jsondataBytes))

	req.Header.Set("Authorization", "Bearer "+Tokens.Slack.API)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("MessageSendError(Slack):\n   %s\n", err.Error())
		return
	}
	resp.Body.Close()

	return
}

func (h *SlackHookBlock) Send() {
	jsondataBytes, _ := json.Marshal(h)

	req, _ := http.NewRequest("POST", SlackMessagePostURI, bytes.NewBuffer(jsondataBytes))

	req.Header.Set("Authorization", "Bearer "+Tokens.Slack.API)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}

	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("MessageSendError(Slack):\n   %s\n", err.Error())
		return
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("%v", errors.Wrapf(err, "ReadAll"))
	}
	if resp.StatusCode != 200 {
		fmt.Printf("Failed Send Slack\n Body: %s\n", body)
	}
	resp.Body.Close()
}

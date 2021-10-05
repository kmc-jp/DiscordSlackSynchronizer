package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

type SlackIndicator struct {
	LastMessages map[string]string
}

func NewSlackIndicator() *SlackIndicator {
	return &SlackIndicator{LastMessages: map[string]string{}}
}

var SlackChatUpdateURL = "https://slack.com/api/chat.update"
var SlackChatDeleteURL = "https://slack.com/api/chat.delete"

func (s *SlackIndicator) Popup(block SlackHookBlock) error {
	_, ok := s.LastMessages[block.Channel]
	if ok {
		s.Remove(block.Channel)
	}

	jsondataBytes, _ := json.Marshal(block)

	req, _ := http.NewRequest("POST", SlackMessagePostURI, bytes.NewBuffer(jsondataBytes))
	req.Header.Set("Authorization", "Bearer "+Tokens.Slack.API)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("MessageSendError(Slack): %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("readall: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed send slack: body: %s", body)
	}
	var r struct {
		OK bool `json:"ok"`
		TS string `json:"ts"`
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("failed send slack: body: %s", body)
	}
	s.LastMessages[block.Channel] = r.TS
	return nil
}

func (s *SlackIndicator) Update(block SlackHookBlock) error {
	ts, ok := s.LastMessages[block.Channel]
	if !ok {
		s.Popup(block)
	}

	type UpdateBlock struct {
		TS string `json:"ts"`
		SlackHookBlock
	}
	updateBlock := &UpdateBlock {
		TS: ts,
		SlackHookBlock: block,
	}

	jsondataBytes, _ := json.Marshal(updateBlock)

	req, _ := http.NewRequest("POST", SlackChatUpdateURL, bytes.NewBuffer(jsondataBytes))
	req.Header.Set("Authorization", "Bearer "+Tokens.Slack.API)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("MessageSendError(Slack): %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("readall: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed send slack: body: %s", body)
	}
	var r struct {
		OK bool `json:"ok"`
		TS string `json:"ts"`
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("failed send slack: body: %s", body)
	}
	s.LastMessages[block.Channel] = r.TS
	return nil
}
func (s *SlackIndicator) Remove(channelID string) error {
	ts, ok := s.LastMessages[channelID]
	if !ok {
		return nil
	}
	delete(s.LastMessages, channelID)

	type RemoveMessage struct {
		Channel string `json:"channel"`
		TS string `json:"ts"`
	}
	removeMessage := &RemoveMessage {
		Channel: channelID,
		TS: ts,
	}
	jsondataBytes, _ := json.Marshal(removeMessage)

	req, _ := http.NewRequest("POST", SlackChatDeleteURL, bytes.NewBuffer(jsondataBytes))
	req.Header.Set("Authorization", "Bearer "+Tokens.Slack.API)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("MessageSendError(Slack): %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("readall: %w", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf("failed delete slack: body: %s", body)
	}
	var r struct {
		OK bool `json:"ok"`
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("failed delete slack: body: %s", body)
	}
	return nil
}
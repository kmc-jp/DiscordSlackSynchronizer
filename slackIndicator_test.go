package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
	"testing"
)

var testComandChannel = "C03875433"

func initSlack() {
	SettingsFile = filepath.Join("settings", "tokens.json")
	b, err := ioutil.ReadFile(SettingsFile)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &Tokens)
	if err != nil {
		panic(err)
	}
	Slack = NewSlackBot(Tokens.Slack.API, Tokens.Slack.Event)
}

func TestPopup(t *testing.T) {
	initSlack()
	slackIndicator := NewSlackIndicator()
	err := slackIndicator.Popup(SlackHookBlock{
		Channel:   testComandChannel,
		Text:      "ok1",
	})
	if err != nil {
		t.Fatal(err)
	}
	(&SlackHookBlock{
		Channel:   testComandChannel,
		Text:      "--",
	}).Send()
	err = slackIndicator.Popup(SlackHookBlock{
		Channel:   testComandChannel,
		Text:      "ok2",
	})
	if err != nil {
		t.Fatal(err)
	}
}
func TestRemove(t *testing.T) {
	initSlack()
	slackIndicator := NewSlackIndicator()
	err := slackIndicator.Popup(SlackHookBlock{
		Channel:   testComandChannel,
		Text:      "removed???",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = slackIndicator.Remove(testComandChannel)
	if err != nil {
		t.Fatal(err)
	}
}
func TestUpdate(t *testing.T) {
	initSlack()
	slackIndicator := NewSlackIndicator()
	err := slackIndicator.Popup(SlackHookBlock{
		Channel:   testComandChannel,
		Text:      "ok?",
	})
	if err != nil {
		t.Fatal(err)
	}
	(&SlackHookBlock{
		Channel:   testComandChannel,
		Text:      "--",
	}).Send()
	err = slackIndicator.Update(SlackHookBlock{
		Channel:   testComandChannel,
		Text:      "updated!",
	})
	if err != nil {
		t.Fatal(err)
	}
}
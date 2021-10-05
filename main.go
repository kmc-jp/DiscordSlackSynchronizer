package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
)

var Tokens struct {
	Slack struct {
		API   string
		Event string
	}
	Discord struct {
		API string
	}
	Gyazo struct {
		API string
	}
}

var SettingsFile string

var Slack *SlackHandler
var Discord *DiscordHandler

func init() {
	SettingsFile = filepath.Join("settings", "tokens.json")
	b, err := ioutil.ReadFile(SettingsFile)
	if err != nil {
		panic(err)
	}
	err = json.Unmarshal(b, &Tokens)
	if err != nil {
		panic(err)
	}
}

func main() {
	if Tokens.Discord.API == "" {
		fmt.Println("No discord token provided. Please run: airhorn -t <bot token>")
		return
	} else {
		Discord = NewDiscordBot(Tokens.Discord.API)
		go Discord.Do()
	}

	Slack = NewSlackBot(Tokens.Slack.API, Tokens.Slack.Event)

	go Slack.Do()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	Discord.Close()
}

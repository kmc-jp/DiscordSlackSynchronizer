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
var Gyazo *GyazoHandler

var slackIndicator *SlackIndicator = NewSlackIndicator()

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
	Gyazo, err = NewGyazoHandler(Tokens.Gyazo.API)
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
		go func() {
			err := Discord.Do()
			if err != nil {
				fmt.Println("Error opening Discord session: ", err)
			}
			// Wait here until CTRL-C or other term signal is received.
			fmt.Println("Discord session is now running.  Press CTRL-C to exit.")
		}()
	}

	Slack = NewSlackBot(Tokens.Slack.API, Tokens.Slack.Event)

	go Slack.Do()

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	Discord.Close()
}

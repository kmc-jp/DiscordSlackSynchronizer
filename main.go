package main

import (
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/configurator"
)

type Token struct {
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

var Tokens Token

var SettingsFile string

var Slack *SlackHandler
var Discord *DiscordHandler
var Gyazo *GyazoHandler

var slackIndicator *SlackIndicator = NewSlackIndicator()

func init() {
	Tokens.Slack.API = os.Getenv("SLACK_API_TOKEN")
	Tokens.Slack.Event = os.Getenv("SLACK_EVENT_TOKEN")
	Tokens.Discord.API = os.Getenv("DISCORD_BOT_TOKEN")
	Tokens.Gyazo.API = os.Getenv("GYAZO_API_TOKEN")
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

	var conf = configurator.New(Tokens.Discord.API, Tokens.Slack.API, filepath.Join("settings", "settings.json"))
	if os.Getenv("SOCK_TYPE") != "" {
		controller, err := conf.Start(os.Getenv("SOCK_TYPE"), os.Getenv("LISTEN_ADDRESS"))
		if err != nil {
			panic(err)
		}

		go func() {
			for command := range controller {
				switch command {
				case configurator.CommandRestart:
					DiscordWebhook = DiscordWebhookType{
						webhookByChannelID: map[string]*discordgo.Webhook{},
						createWebhookLock:  map[string]*sync.RWMutex{},
					}
				}
			}
		}()

	}

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	// Cleanly close down the Discord session.
	Discord.Close()
}

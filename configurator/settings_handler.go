package configurator

import (
	"encoding/json"
	"io/ioutil"
	"net"
	"net/http"
	"os"
)

type SettingsHandler struct {
	confPath string

	Discord *DiscordHandler
	Slack   *SlackHandler

	controller chan int

	Settings []SlackDiscordTable
}

//SlackDiscordTable dict of Channel
type SlackDiscordTable struct {
	Discord       string           `json:"discord_server"`
	Channel       []ChannelSetting `json:"channel"`
	SlackSuffix   string           `json:"slack_suffix"`
	DiscordSuffix string           `json:"discord_suffix"`
}

//ChannelSetting Put send settings
type ChannelSetting struct {
	Comment        string      `json:"comment"`
	SlackChannel   string      `json:"slack"`
	DiscordChannel string      `json:"discord"`
	Setting        SendSetting `json:"setting"`
	Webhook        string      `json:"hook"`
}

//SendSetting put send setting
type SendSetting struct {
	SlackToDiscord           bool `json:"slack2discord"`
	DiscordToSlack           bool `json:"discord2slack"`
	ShowChannelName          bool `json:"ShowChannelName"`
	SendVoiceState           bool `json:"SendVoiceState"`
	SendMuteState            bool `json:"SendMuteState"`
	CreateSlackChannelOnSend bool `json:"CreateSlackChannelOnSend"`
}

func NewSettingsHandler(confPath string, discord *DiscordHandler, slackHandler *SlackHandler) *SettingsHandler {
	return &SettingsHandler{
		confPath:   confPath,
		Discord:    discord,
		Slack:      slackHandler,
		controller: make(chan int),
	}
}

func (s *SettingsHandler) Start(prefix, sock, addr string) (chan int, error) {
	l, err := net.Listen(sock, addr)
	if err != nil {
		return nil, err
	}

	if sock == "unix" {
		err = os.Chmod(addr, 0777)
		if err != nil {
			return nil, err
		}

		defer os.Remove(addr)
	}

	var mux = http.NewServeMux()

	mux.Handle(prefix+"/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, err := ioutil.ReadFile("index.html")
		if err != nil {
			w.Write([]byte("Error: index.html not found"))
			return
		}
		w.Write(b)
	}))
	mux.Handle(prefix+"/api/", s)
	mux.Handle(prefix+"/static/", http.StripPrefix(prefix+"/static/", http.FileServer(http.Dir("static"))))

	err = http.Serve(l, mux)

	return s.controller, err
}

func (s *SettingsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.FormValue("action") {
	case "getDiscordChannels":
		s.GetDiscordChannels(w, r)
	case "getCurrentSettings":
		s.GetCurrentSettings(w, r)
	case "setSettings":
		s.SetSettings(w, r)
	case "getClientInfo":
		s.GetClientInfo(w, r)
	case "getSlackChannels":
		s.GetSlackChannels(w, r)
	case "getDiscordGuildIdentity":
		s.GetDiscordGuildIdentity(w, r)
	default:
		w.Write([]byte("Bad Request"))
		w.WriteHeader(500)
	}
	return
}

func (s *SettingsHandler) ReadSettings() error {
	b, err := ioutil.ReadFile(s.confPath)
	if err != nil {
		return err
	}
	err = json.Unmarshal(b, &s.Settings)
	return err
}

func (s *SettingsHandler) WriteSettings() error {
	b, err := json.MarshalIndent(s.Settings, "", "    ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(s.confPath, b, 0644)
}

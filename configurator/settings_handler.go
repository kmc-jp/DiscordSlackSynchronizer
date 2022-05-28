package configurator

import (
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/kmc-jp/DiscordSlackSynchronizer/settings"
)

type SettingsHandler struct {
	Discord *DiscordHandler
	Slack   *SlackHandler

	controller chan int

	settings *settings.Handler

	socketType     string
	socketFileAddr string
}

func NewSettingsHandler(settings *settings.Handler, discord *DiscordHandler, slackHandler *SlackHandler) *SettingsHandler {
	return &SettingsHandler{
		Discord:  discord,
		Slack:    slackHandler,
		settings: settings,
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
		s.socketFileAddr = addr
		s.socketType = sock
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

	go func() {
		err := http.Serve(l, mux)
		log.Printf("Error: Start http server: %s\n", err.Error())
	}()

	s.controller = make(chan int)

	return s.controller, err
}

func (s *SettingsHandler) Close() (err error) {
	if s.socketType == "unix" {
		err = os.Remove(s.socketFileAddr)
	}
	return
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
}

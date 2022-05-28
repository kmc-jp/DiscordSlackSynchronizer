package configurator

import "github.com/kmc-jp/DiscordSlackSynchronizer/settings"

const (
	CommandRestart = 1 + iota
)

type Handler struct {
	discord struct {
		API string
	}
	slack struct {
		API string
	}

	settings *SettingsHandler
}

func New(discord, slack string) *Handler {
	var handler Handler
	handler.discord.API = discord
	handler.slack.API = slack

	return &handler
}

func (h Handler) Start(prefix, sock, addr string, setting *settings.Handler) (chan int, error) {
	Discord, err := NewDiscordHandler(h.discord.API)
	if err != nil {
		return nil, err
	}

	Slack := NewSlackHandler(h.slack.API)

	h.settings = NewSettingsHandler(
		setting,
		Discord,
		Slack,
	)

	return h.settings.Start(prefix, sock, addr)
}

func (h Handler) Close() error {
	if h.settings == nil {
		return nil
	}
	return h.settings.Close()
}

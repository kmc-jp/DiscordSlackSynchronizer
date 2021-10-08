package configurator

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
	confPath string

	settings *SettingsHandler
}

func New(discord, slack, confPath string) *Handler {
	var handler Handler
	handler.discord.API = discord
	handler.slack.API = slack
	handler.confPath = confPath

	return &handler
}

func (h Handler) Start(prefix, sock, addr string) (chan int, error) {
	Discord, err := NewDiscordHandler(h.discord.API)
	if err != nil {
		return nil, err
	}

	Slack := NewSlackHandler(h.slack.API)

	s := NewSettingsHandler(
		h.confPath,
		Discord,
		Slack,
	)

	h.settings = s

	return s.Start(prefix, sock, addr)
}

func (h Handler) Close() error {
	if h.settings == nil {
		return nil
	}
	return h.settings.Close()
}

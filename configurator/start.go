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
}

func New(discord, slack, confPath string) *Handler {
	var handler Handler
	handler.discord.API = discord
	handler.slack.API = slack
	handler.confPath = confPath

	return &handler
}

func (h Handler) Start(sock, addr string) (chan int, error) {
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

	return s.Start(sock, addr)
}

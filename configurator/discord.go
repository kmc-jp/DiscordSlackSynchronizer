package configurator

import "github.com/bwmarrin/discordgo"

type DiscordHandler struct {
	Session *discordgo.Session
}

func NewDiscordHandler(token string) (*DiscordHandler, error) {
	sess, err := discordgo.New("Bot " + token)
	return &DiscordHandler{sess}, err
}

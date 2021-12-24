package discord_plugin

type defaultPlugin struct{}

func newDefaultPlugin() *defaultPlugin {
	return &defaultPlugin{}
}

func (d *defaultPlugin) GetPrimaryID(discordID string) (string, error) {
	return discordID, nil
}

func (d *defaultPlugin) GetDiscordID(primaryID string) ([]string, error) {
	return []string{primaryID}, nil
}

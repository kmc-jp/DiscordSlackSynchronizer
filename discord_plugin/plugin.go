package discord_plugin

import (
	"os"
	"path/filepath"
	"runtime"
)

var plugin = struct {
	PluginType
}{}

type PluginType interface {
	GetPrimaryID(discordID string) (string, error)
	GetDiscordID(primaryID string) ([]string, error)
}

func init() {
	var path string
	switch runtime.GOOS {
	case "windows":
		path = filepath.Join("plugin", "discord.exe")
	default:
		path = filepath.Join("plugin", "discord")
	}

	_, err := os.Stat(path)
	if err != nil {
		plugin.PluginType = newDefaultPlugin()
		return
	}

	plugin.PluginType = newExternalPlugin(path)
}

func GetPrimaryID(discordID string) (string, error) {
	return plugin.GetPrimaryID(discordID)
}
func GetDiscordID(primaryID string) ([]string, error) {
	return plugin.GetDiscordID(primaryID)
}

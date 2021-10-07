package configurator

import (
	"encoding/json"
	"net/http"
)

func (s SettingsHandler) GetDiscordChannels(w http.ResponseWriter, r *http.Request) {
	guildID := r.FormValue("guild_id")
	channels, err := s.Discord.Session.GuildChannels(guildID)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: GetChannelsError\n" + err.Error()))
		return
	}

	err = json.NewEncoder(w).Encode(channels)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: JsonEncodeError\n" + err.Error()))
		return
	}

	w.Header().Add("Content-type", "application/json")
}

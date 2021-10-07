package configurator

import (
	"encoding/json"
	"net/http"
)

func (s SettingsHandler) GetDiscordGuildIdentity(w http.ResponseWriter, r *http.Request) {
	var guildID = r.FormValue("guild_id")

	if guildID == "" {
		w.WriteHeader(400)
		w.Write([]byte("BadRequest: guild_id not specified"))
		return
	}

	identity, err := s.Discord.Session.Guild(guildID)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: GetIdentityError\n" + err.Error()))
		return
	}

	err = json.NewEncoder(w).Encode(identity)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: JsonMarshalError\n" + err.Error()))
		return
	}

	w.Header().Add("Content-type", "application/json")
}

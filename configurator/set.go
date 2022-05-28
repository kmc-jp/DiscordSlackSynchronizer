package configurator

import (
	"encoding/json"
	"net/http"

	"github.com/kmc-jp/DiscordSlackSynchronizer/settings"
)

func (s *SettingsHandler) SetSettings(w http.ResponseWriter, r *http.Request) {
	var table []settings.SlackDiscordTable

	var err error
	err = json.NewDecoder(r.Body).Decode(&table)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: ParseRequestedSettingsError\n" + err.Error()))
		return
	}

	err = s.settings.WriteChannelMap(table)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: WriteRequestedSettingsError\n" + err.Error()))
		return
	}

	s.controller <- CommandRestart

	w.Write([]byte("OK"))
}

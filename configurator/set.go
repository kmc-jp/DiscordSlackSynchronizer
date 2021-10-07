package configurator

import (
	"encoding/json"
	"net/http"
)

func (s *SettingsHandler) SetSettings(w http.ResponseWriter, r *http.Request) {
	var err error

	err = json.NewDecoder(r.Body).Decode(&s.Settings)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: ParseRequestedSettingsError\n" + err.Error()))
		return
	}

	err = s.WriteSettings()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: WriteRequestedSettingsError\n" + err.Error()))
		return
	}

	s.controller <- CommandRestart

	w.Write([]byte("OK"))

	return
}

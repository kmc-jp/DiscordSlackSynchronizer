package configurator

import (
	"encoding/json"
	"net/http"
)

func (s *SettingsHandler) GetCurrentSettings(w http.ResponseWriter, r *http.Request) {
	var err error
	err = s.ReadSettings()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: ParseSettingsError\n" + err.Error()))
		return
	}

	err = json.NewEncoder(w).Encode(s.Settings)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: JsonEncodeError\n" + err.Error()))
		return
	}

	w.Header().Add("Content-type", "application/json")

	return
}

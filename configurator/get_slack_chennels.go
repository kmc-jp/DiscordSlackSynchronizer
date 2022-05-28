package configurator

import (
	"encoding/json"
	"net/http"
)

func (s *SettingsHandler) GetSlackChannels(w http.ResponseWriter, r *http.Request) {
	channels, err := s.Slack.GetChannels()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: GetChannelsError\n" + err.Error()))
		return
	}

	err = json.NewEncoder(w).Encode(channels)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: JsonMarshalError\n" + err.Error()))
		return
	}

	w.Header().Add("Content-type", "application/json")
}

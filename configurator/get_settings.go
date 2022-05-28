package configurator

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pkg/errors"
)

func (s *SettingsHandler) GetCurrentSettings(w http.ResponseWriter, r *http.Request) {
	table, err := s.settings.GetChannelMap()
	if err != nil {
		var text = errors.Wrap(err, "GetCurrentSettings: GetChannelMap").Error()
		w.WriteHeader(500)
		w.Write([]byte(text))
		log.Println(text)
		return
	}

	err = json.NewEncoder(w).Encode(table)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: JsonEncodeError\n" + err.Error()))
		return
	}

	w.Header().Add("Content-type", "application/json")
}

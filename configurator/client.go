package configurator

import (
	"encoding/json"
	"net/http"
	"os"
)

func (s *SettingsHandler) GetClientInfo(w http.ResponseWriter, r *http.Request) {
	type user struct {
		UserName string
	}
	var resp = user{os.Getenv("REMOTE_USER")}

	err := json.NewEncoder(w).Encode(resp)
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte("InternalServerError: JsonEncodeError"))
		return
	}

	w.Header().Add("Content-type", "application/json")

}

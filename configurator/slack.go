package configurator

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/url"
)

type SlackChannel struct {
	ID                 string            `json:"id"`
	Name               string            `json:"name"`
	IsChannel          bool              `json:"is_channel"`
	IsGroup            bool              `json:"is_group"`
	IsIm               bool              `json:"is_im"`
	Created            int               `json:"created"`
	Creater            string            `json:"creator"`
	IsArchived         bool              `json:"is_archived"`
	IsGeneral          bool              `json:"is_general"`
	Unlinked           int               `json:"unlinked"`
	NameNormalized     string            `json:"name_normalized"`
	IsShared           bool              `json:"is_shared"`
	IsExtShared        bool              `json:"is_ext_shared"`
	IsOrgShared        bool              `json:"is_org_shared"`
	IsPendingExtShared bool              `json:"is_pending_ext_shared"`
	IsMember           bool              `json:"is_member"`
	IsPrivate          bool              `json:"is_private"`
	IsMpim             bool              `json:"is_mpim"`
	Topic              SlackChannelAbout `json:"topic"`
	Purpose            SlackChannelAbout `json:"purpose"`
	NumMembers         int               `json:"num_members"`
}

type SlackChannelAbout struct {
	Value   string `json:"string"`
	Creator string `json:"creator"`
	LastSet int    `json:"last_set"`
}

const SlackAPIEndpoint = "https://slack.com/api/"

type SlackHandler struct {
	token string
}

func NewSlackHandler(token string) *SlackHandler {
	return &SlackHandler{token: token}
}

func (s *SlackHandler) GetChannels() ([]SlackChannel, error) {
	type response struct {
		OK       bool           `json:"ok"`
		Error    string         `json:"error"`
		Channels []SlackChannel `json:"channels"`
		Meta     struct {
			NextCursor string `json:"next_cursor"`
		} `json:"response_metadata"`
	}

	var err error

	var channels []SlackChannel
	var cursor string

	for {
		var body = make(url.Values)
		body.Add("exclude_archived", "true")
		body.Add("limit", "1000")

		if cursor != "" {
			body.Add("cursor", cursor)
		}

		var client = new(http.Client)
		req, err := http.NewRequest("POST", SlackAPIEndpoint+"conversations.list", bytes.NewBufferString(body.Encode()))
		if err != nil {
			return nil, err
		}

		req.Header.Set("Content-type", "application/x-www-form-urlencoded")
		req.Header.Add("Authorization", "Bearer "+s.token)

		resp, err := client.Do(req)

		var apiRes response
		err = json.NewDecoder(resp.Body).Decode(&apiRes)
		if err != nil {
			break
		}

		channels = append(channels, apiRes.Channels...)

		if apiRes.Meta.NextCursor == "" {
			break
		}

		cursor = apiRes.Meta.NextCursor
	}

	return channels, err
}

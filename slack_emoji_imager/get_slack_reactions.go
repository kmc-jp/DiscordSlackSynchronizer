package slack_emoji_imager

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

func (s *Imager) getSlackReactions(channel, timestamp string) (reactions []slackReaction, err error) {
	var requestAttr = make(url.Values)

	requestAttr.Add("channel", channel)
	requestAttr.Add("timestamp", timestamp)

	req, err := http.NewRequest("GET", fmt.Sprintf("%s/reactions.get?%s", SlackAPIEndpoint, requestAttr.Encode()), nil)
	if err != nil {
		return nil, errors.Wrap(err, "MakeNewRequest")
	}

	req.Header.Set("Content-type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", "Bearer "+s.botToken)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Request")
	}
	defer resp.Body.Close()

	var responseAttr reactionsGetResponse
	err = json.NewDecoder(resp.Body).Decode(&responseAttr)
	if err != nil {
		return nil, errors.Wrap(err, "JsonDecode")
	}

	if !responseAttr.OK {
		return nil, errors.Wrap(err, "GetMessageReactions")
	}

	return responseAttr.Message.Reactions, nil
}

package slack_webhook

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
)

// UserProfile contains all the information details of a given user
type UserProfile struct {
	FirstName             string `json:"first_name"`
	LastName              string `json:"last_name"`
	RealName              string `json:"real_name"`
	RealNameNormalized    string `json:"real_name_normalized"`
	DisplayName           string `json:"display_name"`
	DisplayNameNormalized string `json:"display_name_normalized"`
	Email                 string `json:"email,omitempty"`
	Skype                 string `json:"skype,omitempty"`
	Phone                 string `json:"phone,omitempty"`
	Image24               string `json:"image_24"`
	Image32               string `json:"image_32"`
	Image48               string `json:"image_48"`
	Image72               string `json:"image_72"`
	Image192              string `json:"image_192"`
	Image512              string `json:"image_512"`
	ImageOriginal         string `json:"image_original"`
	StatusText            string `json:"status_text,omitempty"`
	StatusEmoji           string `json:"status_emoji,omitempty"`
	StatusExpiration      int    `json:"status_expiration"`
	Team                  string `json:"team"`

	Fields map[string]ProfileLabel `json:"fields"`
}

type ProfileLabel struct {
	Value   string `json:"value"`
	AltText string `json:"alt"`
	Label   string `json:"label"`
}

type UserIdentity struct {
	User string `json:"user"`
	ID   string `json:"id"`
}

type BasicIdentity struct {
	WorkspaceURI string `json:"url"`
	Team         string `json:"team"`
	User         string `json:"user"`
	UserID       string `json:"user_id"`
	TeamID       string `json:"team_id"`
}

func (s Handler) GetUserProfile(user string, includeLabels bool) (profile *UserProfile, err error) {
	var requestAttr = url.Values{}
	if includeLabels {
		requestAttr.Add("include_labels", "true")
	}

	requestAttr.Add("user", user)

	req, err := http.NewRequest("GET", SlackAPIEndpoint+"/users.profile.get?"+requestAttr.Encode(), nil)
	if err != nil {
		return nil, errors.Wrap(err, "Request")
	}
	req.Header.Add("Authorization", "Bearer "+s.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "RequestDo")
	}
	defer resp.Body.Close()

	type responseAttr struct {
		OK      bool        `json:"ok"`
		Error   string      `json:"error"`
		Profile UserProfile `json:"profile"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadBody")
	}

	var response responseAttr

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "UnmarshalJSON")
	}

	if !response.OK {
		return &response.Profile, errors.Wrap(err, response.Error)
	}

	return &response.Profile, nil
}

func (s Handler) SetUserProfile(user string, name string, value string) (profile *UserProfile, err error) {
	var reqBody = new(bytes.Buffer)
	var mw = multipart.NewWriter(reqBody)

	// add name
	mp, err := mw.CreateFormField("name")
	if err != nil {
		return nil, errors.Wrap(err, "CreateFormField(name)")
	}

	mp.Write([]byte(name))

	// add user
	mp, err = mw.CreateFormField("user")
	if err != nil {
		return nil, errors.Wrap(err, "CreateFormField(user)")
	}

	mp.Write([]byte(user))

	mw.Close()

	req, err := http.NewRequest("POST", SlackAPIEndpoint+"/users.profile.set", reqBody)
	if err != nil {
		return nil, errors.Wrap(err, "Request")
	}

	req.Header.Add("Content-Type", mw.FormDataContentType())
	req.Header.Add("Authorization", "Bearer "+s.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "RequestDo")
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadBody")
	}

	type responseAttr struct {
		OK      bool        `json:"ok"`
		Error   string      `json:"error"`
		Profile UserProfile `json:"profile"`
	}

	var response responseAttr

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "UnmarshalJSON")
	}

	if !response.OK {
		return &response.Profile, errors.Wrap(err, response.Error)
	}

	return &response.Profile, nil
}

func (s Handler) GetUserIdentity() (identity *UserIdentity, err error) {
	req, err := http.NewRequest("GET", SlackAPIEndpoint+"/users.identity", nil)
	if err != nil {
		return nil, errors.Wrap(err, "Request")
	}
	req.Header.Add("Authorization", "Bearer "+s.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "RequestDo")
	}
	defer resp.Body.Close()

	type responseAttr struct {
		OK    bool         `json:"ok"`
		Error string       `json:"error"`
		User  UserIdentity `json:"user"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadBody")
	}

	var response responseAttr

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "UnmarshalJSON")
	}

	if !response.OK {
		return nil, errors.Wrap(err, response.Error)
	}

	return &response.User, nil
}

func (s Handler) AuthTest() (*BasicIdentity, error) {
	req, err := http.NewRequest("GET", SlackAPIEndpoint+"/auth.test", nil)
	if err != nil {
		return nil, errors.Wrap(err, "Request")
	}
	req.Header.Add("Authorization", "Bearer "+s.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "RequestDo")
	}
	defer resp.Body.Close()

	type responseAttr struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		*BasicIdentity
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "ReadBody")
	}

	var response responseAttr

	err = json.Unmarshal(body, &response)
	if err != nil {
		return nil, errors.Wrap(err, "UnmarshalJSON")
	}

	if !response.OK {
		return nil, errors.Wrap(err, response.Error)
	}

	return response.BasicIdentity, nil
}

func (u UserProfile) GetUserImageURI() string {
	if u.ImageOriginal != "" {
		return u.ImageOriginal
	}

	if u.Image512 != "" {
		return u.Image512
	}

	if u.Image192 != "" {
		return u.Image192
	}

	if u.Image72 != "" {
		return u.Image72
	}

	if u.Image48 != "" {
		return u.Image48
	}

	if u.Image32 != "" {
		return u.Image32
	}

	if u.Image24 != "" {
		return u.Image24
	}

	return ""
}

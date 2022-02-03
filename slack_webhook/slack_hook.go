package slack_webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

const SlackAPIEndpoint = "https://slack.com/api"

type Handler struct {
	token    string
	Identity BasicIdentity
}

//HookMessage SlackにIncommingWebhook経由のMessage送信形式
type Message struct {
	TS      string `json:"ts,omitempty"`
	Channel string `json:"channel"`

	Type string `json:"type,omitempty"`

	Text        string       `json:"text,omitempty"`
	Blocks      []BlockBase  `json:"blocks,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`

	LinkNames bool   `json:"link_names,omitempty"`
	Username  string `json:"username,omitempty"`
	User      string `json:"user,omitempty"`
	AsUser    bool   `json:"as_user,omitempty"`

	Parse           string `json:"parse,omitempty"`
	ThreadTimestamp string `json:"thread_ts,omitempty"`
	ReplyBroadcast  bool   `json:"reply_broadcast,omitempty"`

	UnfurlLinks bool `json:"unfurl_links"`
	UnfurlMedia bool `json:"unfurl_media"`

	IconURL   string `json:"icon_url,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
}

// Attachment contains all the information for an attachment
type Attachment struct {
	Color    string `json:"color,omitempty"`
	Fallback string `json:"fallback,omitempty"`

	CallbackID string `json:"callback_id,omitempty"`
	ID         int    `json:"id,omitempty"`

	AuthorID      string `json:"author_id,omitempty"`
	AuthorName    string `json:"author_name,omitempty"`
	AuthorSubname string `json:"author_subname,omitempty"`
	AuthorLink    string `json:"author_link,omitempty"`
	AuthorIcon    string `json:"author_icon,omitempty"`

	Title     string `json:"title,omitempty"`
	TitleLink string `json:"title_link,omitempty"`
	Pretext   string `json:"pretext,omitempty"`
	Text      string `json:"text,omitempty"`

	ImageURL string `json:"image_url,omitempty"`
	ThumbURL string `json:"thumb_url,omitempty"`

	ServiceName string `json:"service_name,omitempty"`
	ServiceIcon string `json:"service_icon,omitempty"`
	FromURL     string `json:"from_url,omitempty"`
	OriginalURL string `json:"original_url,omitempty"`

	Fields     []slack.AttachmentField  `json:"fields,omitempty"`
	Actions    []slack.AttachmentAction `json:"actions,omitempty"`
	MarkdownIn []string                 `json:"mrkdwn_in,omitempty"`

	Footer     string `json:"footer,omitempty"`
	FooterIcon string `json:"footer_icon,omitempty"`

	Ts json.Number `json:"ts,omitempty"`
}

type GetConversationHistoryParameters struct {
	ChannelID string `json:"channel"`
	Cursor    string `json:"cursor,omitempty"`
	Inclusive bool   `json:"inclusive,omitempty"`
	Latest    string `json:"latest,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Oldest    string `json:"oldest,omitempty"`
}

type FilesRemoteAddParameters struct {
	Title                 string
	FileType              string
	ExternalID            string
	ExternalURL           string
	IndexableFileContents io.Reader
	PreviewImage          io.Reader
	File                  io.Reader
}

type File struct {
	FileName        string
	Reader          io.Reader
	FileType        string
	InitialComment  string
	ThreadTimestamp string
}

type UnfURLs map[string]UnfURL

type UnfURLsParameters struct {
	Channel          string      `json:"channel"`
	TimeStamp        string      `json:"ts"`
	UnfURLs          UnfURLs     `json:"unfurls"`
	Source           string      `json:"source,omitempty"`
	UnfUrlID         string      `json:"unfurl_id,omitempty"`
	UserAuthBlocks   []BlockBase `json:"user_auth_blocks,omitempty"`
	UserAuthRequired bool        `json:"user_auth_required,omitempty"`
}

type UnfURL struct {
	HideColor bool        `json:"hide_color"`
	Blocks    []BlockBase `json:"blocks"`
}

func New(token string) *Handler {
	var handler = &Handler{token: token}

	identity, _ := handler.AuthTest()
	handler.Identity = *identity

	return handler
}

func (s *Handler) send(jsondataBytes []byte, method string) (string, error) {
	var req *http.Request
	switch method {
	case "update":
		req, _ = http.NewRequest("POST", SlackAPIEndpoint+"/chat.update", bytes.NewBuffer(jsondataBytes))
	case "delete":
		req, _ = http.NewRequest("POST", SlackAPIEndpoint+"/chat.delete", bytes.NewBuffer(jsondataBytes))
	case "send":
		req, _ = http.NewRequest("POST", SlackAPIEndpoint+"/chat.postMessage", bytes.NewBuffer(jsondataBytes))
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("MessageSendError(Slack): %w", err)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("readall: %w", err)
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("failed send slack: body: %s", body)
	}
	var r struct {
		OK bool   `json:"ok"`
		TS string `json:"ts"`
	}
	err = json.Unmarshal(body, &r)
	if err != nil {
		return "", fmt.Errorf("unmarshal: %w", err)
	}
	if !r.OK {
		return "", fmt.Errorf("failed send slack: body: %s", body)
	}

	return r.TS, nil
}

//Send json形式で指定したURLにPOSTする。
func (s *Handler) Send(message Message) (string, error) {
	jsonDataBytes, _ := json.Marshal(message)
	return s.send(jsonDataBytes, "send")
}

func (s *Handler) Update(message Message) (string, error) {
	jsonDataBytes, _ := json.Marshal(message)
	return s.send(jsonDataBytes, "update")
}

func (s Handler) Remove(channel, ts string) (string, error) {
	type RemoveMessage struct {
		Channel string `json:"channel"`
		TS      string `json:"ts"`
	}
	var removeMessage = &RemoveMessage{
		Channel: channel,
		TS:      ts,
	}
	jsonDataBytes, _ := json.Marshal(removeMessage)
	return s.send(jsonDataBytes, "delete")
}

func (s *Handler) GetMessages(channelID, timestamp string, limit int) ([]Message, error) {
	var requestAttr = url.Values{}

	requestAttr.Set("channel", channelID)
	requestAttr.Set("latest", timestamp)
	requestAttr.Set("limit", strconv.Itoa(limit))
	requestAttr.Set("inclusive", "true")

	var hasMore = true
	var messages = []Message{}
	var cursor string

	// get all messages while Slack sends "has_more" message
	for hasMore {
		if len(messages) >= limit {
			break
		}
		requestAttr.Set("cursor", cursor)

		req, _ := http.NewRequest("GET", SlackAPIEndpoint+"/conversations.history?"+requestAttr.Encode(), nil)

		req.Header.Set("Authorization", "Bearer "+s.token)
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("MessageSendError(Slack): %w", err)
		}
		defer resp.Body.Close()
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("readall: %w", err)
		}
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("failed send slack: body: %s", body)
		}

		var r struct {
			OK           bool      `json:"ok"`
			Messages     []Message `json:"messages"`
			HasMore      bool      `json:"has_more"`
			ResponseMeta struct {
				NextCursor string `json:"next_cursor"`
			} `json:"response_metadata"`
		}

		err = json.Unmarshal(body, &r)
		if err != nil {
			return nil, fmt.Errorf("unmarshal: %w", err)
		}
		if !r.OK {
			return nil, fmt.Errorf("failed send slack: body: %s", body)
		}

		if err != nil {
			return nil, err
		}

		messages = append(messages, r.Messages...)
		hasMore = r.HasMore
		cursor = r.ResponseMeta.NextCursor
	}

	return messages, nil
}

func (s *Handler) GetMessage(channelID, timestamp string) (*Message, error) {
	msgs, err := s.GetMessages(channelID, timestamp, 1)
	if len(msgs) < 1 {
		return nil, errors.Wrap(err, "NotFound")
	}
	return &msgs[0], err
}

func (s *Handler) FilesUpload(file File, channels ...string) (*slack.File, error) {
	var body = new(bytes.Buffer)

	var mw = multipart.NewWriter(body)

	if file.FileName != "" {
		pw, err := mw.CreateFormField("filename")
		if err != nil {
			return nil, errors.Wrap(err, "CreatingPartAtFileName")
		}

		pw.Write([]byte(file.FileName))
	}

	if file.FileType != "" {
		pw, err := mw.CreateFormField("filetype")
		if err != nil {
			return nil, errors.Wrap(err, "CreatingPartAtFileType")
		}

		pw.Write([]byte(file.FileType))
	}

	if file.InitialComment != "" {
		pw, err := mw.CreateFormField("initial_comment")
		if err != nil {
			return nil, errors.Wrap(err, "CreatingPartAtInitialComment")
		}

		pw.Write([]byte(file.InitialComment))
	}

	if len(channels) > 0 {
		pw, err := mw.CreateFormField("channels")
		if err != nil {
			return nil, errors.Wrap(err, "CreatingPartAtInitialComment")
		}

		pw.Write([]byte(strings.Join(channels, ",")))
	}

	if file.ThreadTimestamp != "" {
		pw, err := mw.CreateFormField("thread_ts")
		if err != nil {
			return nil, errors.Wrap(err, "CreatingPartAtThreadTimestamp")
		}

		pw.Write([]byte(file.ThreadTimestamp))
	}

	pw, err := mw.CreateFormField("content")
	if err != nil {
		return nil, errors.Wrap(err, "CreatingPartAtFile")
	}

	io.Copy(pw, file.Reader)

	mw.Close()

	var req *http.Request
	var reqURI = fmt.Sprintf("%s/files.upload", SlackAPIEndpoint)

	req, err = http.NewRequest("POST", reqURI, body)
	if err != nil {
		return nil, errors.Wrap(err, "Requrst")
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Sending")
	}
	defer resp.Body.Close()

	type responseAttrType struct {
		OK    bool       `json:"ok"`
		Error string     `json:"error"`
		File  slack.File `json:"file"`
	}

	var responseAttr responseAttrType

	err = func() error {
		defer func() {
			err := recover()
			if err != nil {
				log.Printf("Panic in Decoding JSON: %v\n", err)
			}
		}()
		return json.NewDecoder(resp.Body).Decode(&responseAttr)
	}()

	if err != nil {
		return &responseAttr.File, errors.Wrap(err, "DecodingJSON")
	}

	if !responseAttr.OK {
		return &responseAttr.File, errors.New("SlackAPIError: " + responseAttr.Error)
	}

	return &responseAttr.File, nil
}

func (s *Handler) ChatUnfURL(parameters UnfURLsParameters) error {
	var body = new(bytes.Buffer)
	err := json.NewEncoder(body).Encode(parameters)
	if err != nil {
		return errors.Wrap(err, "JsonEncode")
	}

	var reqURI = fmt.Sprintf("%s/chat.unfurl", SlackAPIEndpoint)
	req, err := http.NewRequest("POST", reqURI, body)
	if err != nil {
		return errors.Wrap(err, "Requrst")
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "Request")
	}

	type responseAttrType struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
	}

	var responseAttr responseAttrType
	err = func() error {
		defer func() {
			err := recover()
			if err != nil {
				log.Printf("Error At Json Decoding: %v", err)
			}
		}()
		return json.NewDecoder(resp.Body).Decode(&responseAttr)
	}()

	if err != nil {
		return errors.Wrap(err, "JsonDecode")
	}

	if !responseAttr.OK {
		return errors.New(fmt.Sprintf("ErrorAtUnfurlAPI: %s", responseAttr.Error))
	}

	return nil
}

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
)

type DiscordReactionHandler struct {
	token string

	gyazo          *GyazoHandler
	reactionImager ReactionImagerType
}

type ReactionImagerType interface {
	AddEmoji(name string, uri string)
	RemoveEmoji(string)
	MakeReactionsImage(channel string, timestamp string) (r io.Reader, err error)
	GetEmojiURI(name string) string
}

type DiscordMessage struct {
	*discordgo.Message
	Attachments []DiscordAttachment `json:"attachments"`
}

type DiscordAttachment struct {
	URL         string `json:"url,omitempty"`
	Description string `json:"description,omitempty"`
	ID          int    `json:"id"`
	ProxyURL    string `json:"proxy_url,omitempty"`
	Filename    string `json:"filename,omitempty"`
	Width       int    `json:"width,omitempty"`
	Height      int    `json:"height,omitempty"`
	Size        int    `json:"size,omitempty"`
}

type DiscordFile struct {
	FileName    string
	Reader      io.Reader
	ContentType string
}

const DiscordReactionThreadName = "SlackEmoji"

func NewDiscordReactionHandler(token string, gyazo *GyazoHandler) *DiscordReactionHandler {
	return &DiscordReactionHandler{
		token: token,
		gyazo: gyazo,
	}
}

func (d *DiscordReactionHandler) AddReactionImager(imager ReactionImagerType) {
	d.reactionImager = imager
}

func (d *DiscordReactionHandler) GetReaction(channel string, timestamp string) error {
	const ReactionGifName = "reactions.gif"

	var zeroReaction bool

	r, err := d.reactionImager.MakeReactionsImage(channel, timestamp)
	switch err {
	case nil:
		break
	case SlackEmojiImagerErrorNoReactions:
		zeroReaction = true
	default:
		return err
	}

	var cs, _ = findDiscordChannel(channel)
	if !cs.Setting.SlackToDiscord {
		return nil
	}

	messages, err := d.getMessages(cs.DiscordChannel, timestamp)
	if err != nil {
		return err
	}

	// TODO: choose the correct message

	var unixTimeStamp = strings.Split(timestamp, ".")

	var i int
	var unixSec int64
	var numChar = len([]rune(unixTimeStamp[0]))
	for _, nstr := range unixTimeStamp[0] {

		num, err := strconv.Atoi(string(nstr))
		if err != nil {
			return err
		}

		unixSec += int64(float64(num) * math.Pow10(numChar-i-1))
		i++
	}

	unixNanoSec, err := strconv.Atoi(unixTimeStamp[0])
	if err != nil {
		return err
	}

	var message DiscordMessage

	srcTime := time.Unix(unixSec, int64(unixNanoSec))
	for i, msg := range messages {
		t, err := msg.Timestamp.Parse()
		if err != nil {
			continue
		}

		if srcTime.UnixMilli() > t.UnixMilli() {
			if i == 0 {
				return fmt.Errorf("illigal time")
			}

			message.Message = &messages[i-1]
			break
		}
	}
	if message.Message == nil {
		return fmt.Errorf("MessageNotFound")
	}

	message.Attachments = make([]DiscordAttachment, 0)

	for i := range message.Message.Attachments {
		if message.Message.Attachments[i] == nil {
			continue
		}

		var oldAtt = message.Message.Attachments[i]
		if strings.HasSuffix(oldAtt.URL, ReactionGifName) {
			continue
		}

		id, err := strconv.Atoi(oldAtt.ID)
		if err != nil {
			continue
		}

		message.Attachments = append(message.Attachments, DiscordAttachment{
			URL:      oldAtt.URL,
			ID:       id,
			ProxyURL: oldAtt.ProxyURL,
			Filename: oldAtt.Filename,
			Width:    oldAtt.Width,
			Height:   oldAtt.Height,
			Size:     oldAtt.Size,
		})

	}

	if zeroReaction {
		return d.editMessage(message.ChannelID, message.ID, message, []DiscordFile{})
	}

	var file = DiscordFile{
		FileName:    ReactionGifName,
		Reader:      r,
		ContentType: "image/gif",
	}

	return d.editMessage(message.ChannelID, message.ID, message, []DiscordFile{file})
}

func (d *DiscordReactionHandler) deleteMessage(channelID, messageID string) (err error) {
	var requestAttr = make(url.Values)

	requestAttr.Set("name", DiscordReactionThreadName)
	requestAttr.Set("auto_archive_duration", "10")

	var client = http.DefaultClient
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/channels/%s/messages/%s",
			DiscordAPIEndpoint, channelID, messageID,
		),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+d.token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return nil
}

func (d *DiscordReactionHandler) editMessage(channelID, messageID string, message DiscordMessage, files []DiscordFile) (err error) {
	var hook = DiscordWebhook.Get(channelID)

	if err != nil {
		return
	}

	var body = new(bytes.Buffer)

	var mw = multipart.NewWriter(body)

	var mh = make(textproto.MIMEHeader)
	mh.Set("Content-Type", "application/json")
	mh.Set(`Content-Disposition`, `form-data; name="payload_json"`)

	pw, err := mw.CreatePart(mh)
	if err != nil {
		return err
	}

	b, err := json.MarshalIndent(message, "", "    ")

	var jsonBuf = bytes.NewBuffer(b)
	io.Copy(pw, jsonBuf)

	for i, file := range files {
		var mh = make(textproto.MIMEHeader)
		mh.Set("Content-Type", file.ContentType)
		mh.Set(`Content-Disposition`, fmt.Sprintf(`form-data; name="files[%d]"; filename="%s"`, i, file.FileName))

		pw, err := mw.CreatePart(mh)
		if err != nil {
			return err
		}

		io.Copy(pw, file.Reader)
	}

	mw.Close()

	req, err := http.NewRequest(
		"PATCH",
		fmt.Sprintf("%s/webhooks/%s/%s/messages/%s",
			DiscordAPIEndpoint, hook.ID, hook.Token, messageID,
		),
		body,
	)
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	return
}

func (d *DiscordReactionHandler) writePartValue(mw *multipart.Writer, name, value string) error {
	pw, err := mw.CreateFormField(name)
	if err != nil {
		return err
	}

	pw.Write([]byte(value))

	return nil
}

func (d *DiscordReactionHandler) createMessage(channelID, content string) (Err error) {

	var requestAttr = make(url.Values)

	requestAttr.Set("name", DiscordReactionThreadName)
	requestAttr.Set("auto_archive_duration", "10")

	var client = http.DefaultClient
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/channels/%s/messages",
			DiscordAPIEndpoint, channelID,
		),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+d.token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var responseAttr []discordgo.Channel
	err = json.NewDecoder(resp.Body).Decode(&responseAttr)
	if err != nil {
		return
	}

	return
}

func (d *DiscordReactionHandler) findOwnThread(guildID, channelID string) (channels []discordgo.Channel, err error) {
	var client = http.DefaultClient
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/guilds/%s/threads/active",
			DiscordAPIEndpoint, channelID,
		),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+d.token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	type responseAttrType struct {
		Threads []discordgo.Channel `json:"threads"`
		Members []discordgo.Member  `json:"members"`
	}

	var responseAttr responseAttrType
	err = json.NewDecoder(resp.Body).Decode(&responseAttr)
	if err != nil {
		return
	}

	channels = make([]discordgo.Channel, 0)

	for _, ch := range responseAttr.Threads {
		if ch.ID == channelID {
			channels = append(channels, ch)
		}
	}

	return channels, nil
}

func (d *DiscordReactionHandler) startThread(channelID, messageID string) (channel discordgo.Channel, err error) {

	var requestAttr = make(url.Values)

	requestAttr.Set("name", DiscordReactionThreadName)
	requestAttr.Set("auto_archive_duration", "10")

	var client = http.DefaultClient
	req, err := http.NewRequest(
		"POST",
		fmt.Sprintf("%s/channels/%s/messages/%s/threads?%s",
			DiscordAPIEndpoint, channelID, messageID, requestAttr.Encode(),
		),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+d.token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var responseAttr []discordgo.Channel
	err = json.NewDecoder(resp.Body).Decode(&responseAttr)
	if err != nil {
		return
	}

	return channel, nil
}

func (d *DiscordReactionHandler) getMessages(channelID, timestamp string) (messages []discordgo.Message, err error) {
	// convert timestamp to message id
	const DiscordEpoc = 1420070400000
	if !strings.Contains(timestamp, ".") {
		return
	}

	var unixTimeStamp = strings.Split(timestamp, ".")

	var i int
	var unixSec int64
	var numChar = len([]rune(unixTimeStamp[0]))
	for _, nstr := range unixTimeStamp[0] {
		num, err := strconv.Atoi(string(nstr))
		if err != nil {
			return nil, fmt.Errorf("TimeParseError: %s", err.Error())
		}

		unixSec += int64(float64(num) * math.Pow10(numChar-i-1))
		i++
	}

	unixNanoSec, err := strconv.Atoi(unixTimeStamp[0])
	if err != nil {
		return
	}

	var discordTimeStamp = unixSec*1000 + int64(unixNanoSec)/1000 - DiscordEpoc

	var messageIDint64 int64 = discordTimeStamp
	var messageID string

	messageIDint64 <<= 22
	messageID = fmt.Sprintf("%v", messageIDint64)

	fmt.Println(messageID)

	var requestAttr = make(url.Values)

	requestAttr.Set("around", messageID)
	requestAttr.Set("limit", "100")

	var client = http.DefaultClient
	req, err := http.NewRequest(
		"GET",
		fmt.Sprintf("%s/channels/%s/messages?%s", DiscordAPIEndpoint, channelID, requestAttr.Encode()),
		nil,
	)
	if err != nil {
		return
	}

	req.Header.Set("Authorization", "Bot "+d.token)

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	fmt.Println(resp.Body)

	var responseAttr []discordgo.Message
	err = func() error {
		defer func() {
			err := recover()
			log.Println(err)
		}()
		return json.NewDecoder(resp.Body).Decode(&responseAttr)
	}()
	if err != nil {
		return
	}

	return responseAttr, nil
}

func (d *DiscordReactionHandler) AddEmoji(name, value string) {
	d.reactionImager.AddEmoji(name, value)
}

func (d *DiscordReactionHandler) RemoveEmoji(name string) {
	d.reactionImager.RemoveEmoji(name)
}

func (d *DiscordReactionHandler) GetEmojiURI(name string) string {
	return d.reactionImager.GetEmojiURI(name)
}

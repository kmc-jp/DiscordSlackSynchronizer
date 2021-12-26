package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/discord_webhook"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_webhook"
	"github.com/pkg/errors"
	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	scm "github.com/slack-go/slack/socketmode"
)

type SlackLastMessages map[string]string

type SlackHandler struct {
	api     *slack.Client
	userAPI *slack.Client
	scm     *scm.Client
	regExp  struct {
		UserID  *regexp.Regexp
		Channel *regexp.Regexp
		URI     *regexp.Regexp
	}

	gyazo *GyazoHandler

	messageUnescaper *strings.Replacer

	discordHook *discord_webhook.Handler
	hook        *slack_webhook.Handler

	apiToken   string
	eventToken string
	userToken  string

	workspaceURI string

	reactionHandler ReactionHandler
}

func NewSlackBot(apiToken, eventToken string) *SlackHandler {
	var slackBot SlackHandler

	slackBot.api = slack.New(
		apiToken,
		slack.OptionAppLevelToken(eventToken),
	)
	slack.OptionLog(log.New(os.Stdout, "slack-bot: ", log.Lshortfile|log.LstdFlags))

	slackBot.scm = scm.New(slackBot.api)

	slackBot.regExp.UserID = regexp.MustCompile(`<@(\S+)>`)
	slackBot.regExp.Channel = regexp.MustCompile(`<#(\S+)\|(\S+)>`)
	slackBot.regExp.URI = regexp.MustCompile(`<(https??://\S+)\|(\S+)>`)

	slackBot.apiToken = apiToken
	slackBot.eventToken = eventToken

	res, _ := slackBot.api.AuthTest()
	slackBot.workspaceURI = res.URL

	slackBot.messageUnescaper = strings.NewReplacer(
		"&amp;", "&",
		"&lt;", "<",
		"&gt;", ">",
	)

	return &slackBot
}

func (s *SlackHandler) Do() {
	go func() {
		var err = s.scm.Run()
		if err != nil {
			fmt.Println(err)
		}
	}()

	for ev := range s.scm.Events {
		switch ev.Type {
		case scm.EventTypeConnected:
			fmt.Printf("Start websocket connection with Slack\n")
		case scm.EventTypeEventsAPI:
			s.scm.Ack(*ev.Request)

			evp, ok := ev.Data.(slackevents.EventsAPIEvent)
			if !ok {
				continue
			}
			switch evp.Type {
			case slackevents.CallbackEvent:
				switch evi := evp.InnerEvent.Data.(type) {
				case *slackevents.AppMentionEvent:
				case *slackevents.MessageEvent:
					s.messageHandle(evi)
				case *slackevents.EmojiChangedEvent:
					s.emojiChangeHandle(evi)
				case *slackevents.ReactionAddedEvent:
					if evi.Item.Type == "message" {
						s.reactionHandle(evi.Item.Channel, evi.Item.Timestamp)
					}
				case *slackevents.ReactionRemovedEvent:
					if evi.Item.Type == "message" {
						s.reactionHandle(evi.Item.Channel, evi.Item.Timestamp)
					}
				}
			}

		}
	}
}

func (s *SlackHandler) SetGyazoHandler(g *GyazoHandler) {
	s.gyazo = g
}

func (s *SlackHandler) SetReactionHandler(handler ReactionHandler) {
	s.reactionHandler = handler
}

func (s *SlackHandler) SetDiscordWebhook(hook *discord_webhook.Handler) {
	s.discordHook = hook
}

func (s *SlackHandler) SetSlackWebhook(hook *slack_webhook.Handler) {
	s.hook = hook
}

func (s *SlackHandler) SetUserToken(token string) {
	s.userToken = token
	s.userAPI = slack.New(token)
}

func (s *SlackHandler) reactionHandle(channel string, timestamp string) {
	if s.reactionHandler != nil {
		err := s.reactionHandler.GetReaction(channel, timestamp)
		if err != nil {
			log.Println(err)
		}
	}
}

func (s *SlackHandler) emojiChangeHandle(ev *slackevents.EmojiChangedEvent) {
	switch ev.Subtype {
	case "add":
		var uri = ev.Value
		if strings.HasPrefix(ev.Value, "alias:") {
			uri = s.reactionHandler.GetEmojiURI(strings.TrimPrefix(uri, "alias:"))
		}
		s.reactionHandler.AddEmoji(ev.Name, uri)
	case "remove":
		for _, name := range ev.Names {
			s.reactionHandler.RemoveEmoji(name)
		}
	case "rename":
		s.reactionHandler.RemoveEmoji(ev.OldName)
		s.reactionHandler.AddEmoji(ev.NewName, ev.Value)
	}
}

func (s *SlackHandler) messageHandle(ev *slackevents.MessageEvent) {
	var cs, discordID = findDiscordChannel(ev.Channel)
	//Confirm Slack to Discord setting
	if !cs.Setting.SlackToDiscord {
		return
	}

	var fileURL []string

	if s.gyazo != nil {
		for _, f := range ev.Files {
			if f.Filetype == "png" || f.Filetype == "jpg" || f.Filetype == "gif" {
				req, err := http.NewRequest("GET", f.URLPrivate, nil)

				req.Header.Set("Authorization", "Bearer "+s.apiToken)
				if err != nil {
					continue
				}

				client := new(http.Client)
				resp, err := client.Do(req)
				if err != nil {
					continue
				}
				defer resp.Body.Close()

				image, err := s.gyazo.Upload(resp.Body)
				if err != nil {
					fileURL = append(fileURL, f.URLPrivate)
					continue
				}
				fileURL = append(fileURL, image.PermalinkURL+"."+f.Filetype)
			} else {
				fileURL = append(fileURL, f.URLPrivate)
			}

		}
	}

	text, err := s.EscapeMessage(ev.Text)
	if err != nil {
		return
	}

	user, err := s.api.GetUserInfo(ev.User)
	if err != nil {
		return
	}

	var name = user.Profile.DisplayName
	if name == "" {
		name = user.RealName
	}

	var message = discord_webhook.Message{
		AvaterURL: user.Profile.ImageOriginal,
		UserName:  name,
		Message: &discordgo.Message{
			GuildID:   discordID,
			ChannelID: cs.DiscordChannel,
			Content:   text,
		},
	}

	// Send by webhook
	newMessage, err := s.discordHook.Send(message.ChannelID, message.ID, message, true, nil)
	if err != nil {
		log.Println(errors.Wrap(err, "ResendingMessage: "))
		return
	}

	if s.userAPI != nil {
		_, _, err = s.userAPI.DeleteMessage(ev.Channel, ev.TimeStamp)
		if err == nil {
			var message = slack_webhook.Message{
				IconURL:     user.Profile.ImageOriginal,
				Username:    name,
				Channel:     ev.Channel,
				Text:        fmt.Sprintf("%s <%s%s|%s>", ev.Text, SlackMessageDummyURI, newMessage.Timestamp, "ㅤ"),
				UnfurlLinks: true,
				UnfurlMedia: true,
				LinkNames:   true,
			}

			// Send message to Slack
			s.hook.Send(message)
		}
	}

	for _, f := range fileURL {
		message = discord_webhook.Message{
			AvaterURL: user.Profile.ImageOriginal,
			UserName:  name,
			Message: &discordgo.Message{
				GuildID:   discordID,
				ChannelID: cs.DiscordChannel,
				Content:   f,
			},
		}

		s.discordHook.Send(message.ChannelID, message.ID, message, false, nil)
	}
}

func (s *SlackHandler) EscapeMessage(content string) (output string, err error) {
	for _, id := range s.regExp.UserID.FindAllStringSubmatch(content, -1) {
		if len(id) < 2 {
			continue
		}

		u, err := s.api.GetUserInfo(id[1])
		if err != nil {
			return "", err
		}

		var repl = u.Profile.DisplayName
		if repl == "" {
			repl = u.RealName
		}
		content = strings.Join(strings.Split(content, id[0]), "`@"+repl+"`")
	}

	for _, ch := range s.regExp.Channel.FindAllStringSubmatch(content, -1) {
		if len(ch) < 3 {
			continue
		}

		content = strings.Join(strings.Split(content,
			fmt.Sprintf("<#%s|%s>", ch[1], ch[2])),
			fmt.Sprintf("`#%s`(URI: <%sarchives/%s>)", ch[2], s.workspaceURI, ch[1]),
		)
	}

	for _, uri := range s.regExp.URI.FindAllStringSubmatch(content, -1) {
		if len(uri) < 3 {
			continue
		}

		if uri[1] == uri[2] {
			content = strings.Join(strings.Split(content,
				fmt.Sprintf("<%s|%s>", uri[1], uri[2])),
				fmt.Sprintf("<%s>)", uri[1]),
			)
		}
		content = strings.Join(strings.Split(content,
			fmt.Sprintf("<%s|%s>", uri[1], uri[2])),
			fmt.Sprintf("%s(URI: <%s>)", uri[2], uri[1]),
		)
	}

	return s.messageUnescaper.Replace(content), nil
}

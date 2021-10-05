package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	scm "github.com/slack-go/slack/socketmode"
)

type SlackHandler struct {
	api    *slack.Client
	scm    *scm.Client
	regExp struct {
		UserID  *regexp.Regexp
		Channel *regexp.Regexp
		URI     *regexp.Regexp
	}

	workspaceURI string
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

	res, _ := slackBot.api.AuthTest()
	slackBot.workspaceURI = res.URL

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

			evp, _ := ev.Data.(slackevents.EventsAPIEvent)
			switch evp.Type {
			case slackevents.CallbackEvent:
				switch evi := evp.InnerEvent.Data.(type) {
				case *slackevents.AppMentionEvent:
				case *slackevents.MessageEvent:
					s.messageHandle(evi)
				}
			}
		}
	}

}

func (s *SlackHandler) messageHandle(ev *slackevents.MessageEvent) {
	var cs, discordID = findDiscordChannel(ev.Channel)
	//Confirm Slack to Discord setting
	if !cs.Setting.SlackToDiscord {
		return
	}
	var fileURL []string
	for _, f := range ev.Files {
		if f.Filetype == "png" || f.Filetype == "jpg" || f.Filetype == "gif" {
			req, _ := http.NewRequest("GET", f.URLPrivate, nil)
			req.Header.Set("Authorization", "Bearer "+Tokens.Slack.API)

			client := new(http.Client)
			resp, err := client.Do(req)
			if err != nil {
				continue
			}

			defer resp.Body.Close()
			image, err := Gyazo.Upload(resp.Body)
			if err != nil {
				fileURL = append(fileURL, f.URLPrivate)
				continue
			}
			fileURL = append(fileURL, image.PermalinkURL+"."+f.Filetype)
		} else {
			fileURL = append(fileURL, f.URLPrivate)
		}

	}

	var text string = ev.Text

	for _, id := range s.regExp.UserID.FindAllStringSubmatch(ev.Text, -1) {
		if len(id) < 2 {
			continue
		}

		u, err := s.api.GetUserInfo(id[1])
		if err != nil {
			return
		}

		var repl = u.Profile.DisplayName
		if repl == "" {
			repl = u.RealName
		}
		text = strings.Join(strings.Split(text, id[0]), "`@"+repl+"`")
	}

	for _, ch := range s.regExp.Channel.FindAllStringSubmatch(ev.Text, -1) {
		if len(ch) < 3 {
			continue
		}

		text = strings.Join(strings.Split(text,
			fmt.Sprintf("<#%s|%s>", ch[1], ch[2])),
			fmt.Sprintf("`#%s`(URI: <%sarchives/%s>)", ch[2], s.workspaceURI, ch[1]),
		)
	}

	for _, uri := range s.regExp.URI.FindAllStringSubmatch(ev.Text, -1) {
		if len(uri) < 3 {
			continue
		}

		if uri[1] == uri[2] {
			text = strings.Join(strings.Split(text,
				fmt.Sprintf("<%s|%s>", uri[1], uri[2])),
				fmt.Sprintf("<%s>)", uri[1]),
			)
		}
		text = strings.Join(strings.Split(text,
			fmt.Sprintf("<%s|%s>", uri[1], uri[2])),
			fmt.Sprintf("%s(URI: <%s>)", uri[2], uri[1]),
		)
	}

	user, err := s.api.GetUserInfo(ev.User)
	if err != nil {
		return
	}

	var name = user.Profile.DisplayName
	if name == "" {
		name = user.RealName
	}

	var message = DiscordHookMessage{
		Channel: cs.DiscordChannel,
		Server:  discordID,
		Name:    name,
		IconURL: user.Profile.ImageOriginal,
		Text:    text,
	}

	message.Send()

	for _, f := range fileURL {
		message = DiscordHookMessage{
			Channel: cs.DiscordChannel,
			Server:  discordID,
			Name:    name,
			IconURL: user.Profile.ImageOriginal,
			Text:    f,
		}
		message.Send()
	}
}

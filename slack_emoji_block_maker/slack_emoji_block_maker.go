package slack_emoji_block_maker

import (
	"fmt"
	"strconv"

	"github.com/bwmarrin/discordgo"
	"github.com/kmc-jp/DiscordSlackSynchronizer/slack_webhook"
)

const DiscordEmojiEndpoint = "https://cdn.discordapp.com/emojis"

func Build(reacts []*discordgo.MessageReactions) []slack_webhook.BlockBase {
	var blocks = []slack_webhook.BlockBase{}
	var elements = []slack_webhook.BlockElement{}

	var emojiCount int
	for _, react := range reacts {
		if react.Emoji.ID == "" {
			var text = react.Emoji.Name
			var stdEmojiElem = slack_webhook.MrkdwnElement(text, false)

			elements = append(elements, stdEmojiElem)
		} else {
			var imageURI string
			switch react.Emoji.Animated {
			case true:
				// is GIF
				imageURI = fmt.Sprintf("%s/%s.gif", DiscordEmojiEndpoint, react.Emoji.ID)
			case false:
				// is PNG
				imageURI = fmt.Sprintf("%s/%s.png", DiscordEmojiEndpoint, react.Emoji.ID)
			}

			var ctmEmojiElem = slack_webhook.ImageElement(imageURI, react.Emoji.Name)
			elements = append(elements, ctmEmojiElem)
		}

		var countElem = slack_webhook.MrkdwnElement(strconv.Itoa(react.Count), false)
		elements = append(elements, countElem)

		emojiCount++
		if emojiCount%4 == 0 {
			var block = slack_webhook.ContextBlock(elements...)
			blocks = append(blocks, block)

			elements = []slack_webhook.BlockElement{}
		}
	}

	if emojiCount%4 > 0 {
		var block = slack_webhook.ContextBlock(elements...)
		blocks = append(blocks, block)
	}

	return blocks
}

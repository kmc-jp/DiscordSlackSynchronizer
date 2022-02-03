package discord_webhook

import "github.com/bwmarrin/discordgo"

func FromDiscordgoMessage(dMessage *discordgo.Message) Message {
	var message = Message{
		// Components  []Component  `json:"components,omitempty"`
		// Attachments []Attachment `json:"attachments"`
		UserName:  dMessage.Author.Username,
		AvaterURL: dMessage.Author.AvatarURL(""),

		ID:               dMessage.ID,
		ChannelID:        dMessage.ChannelID,
		GuildID:          dMessage.GuildID,
		Content:          dMessage.Content,
		Timestamp:        dMessage.Timestamp,
		EditedTimestamp:  dMessage.EditedTimestamp,
		MentionRoles:     dMessage.MentionRoles,
		TTS:              dMessage.TTS,
		MentionEveryone:  dMessage.MentionEveryone,
		Author:           dMessage.Author,
		Embeds:           dMessage.Embeds,
		Mentions:         dMessage.Mentions,
		Reactions:        dMessage.Reactions,
		Pinned:           dMessage.Pinned,
		Type:             dMessage.Type,
		WebhookID:        dMessage.WebhookID,
		Member:           dMessage.Member,
		MentionChannels:  dMessage.MentionChannels,
		Activity:         dMessage.Activity,
		Application:      dMessage.Application,
		MessageReference: dMessage.MessageReference,
		Flags:            dMessage.Flags,
	}

	message.Attachments = make([]Attachment, 0)
	for _, attach := range dMessage.Attachments {
		if attach == nil {
			continue
		}
		message.Attachments = append(message.Attachments, Attachment{
			URL:      attach.URL,
			ID:       attach.ID,
			ProxyURL: attach.ProxyURL,
			Filename: attach.Filename,
			Width:    attach.Width,
			Height:   attach.Height,
			Size:     attach.Size,
		})
	}

	return message
}

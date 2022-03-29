package main

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestVoiceState(t *testing.T) {
	voiceChannels := VoiceChannels{}

	memberID1 := "wass80"
	member1 := &discordgo.Member{User: &discordgo.User{ID: memberID1, Username: "user80"}}

	channelID1 := "general"
	channel1 := &discordgo.Channel{ID: channelID1, Name: "一般"}

	// Join
	exists := voiceChannels.Join(channel1, member1)
	if exists {
		t.Fatal("Expected the user not already joined the channel")
	}
	if len(voiceChannels.Channels[channelID1].Users) != 1 {
		t.Fatalf("Expected 1 user in voice channel, But %v",
			voiceChannels.Channels[channelID1].Users)
	}

	channel, ok := voiceChannels.FindChannelHasUser(memberID1)
	if !ok {
		t.Fatalf("Expected find the user")
	}
	if channel != channelID1 {
		t.Fatalf("Expected channelID %s, but got %s", channelID1, channel)
	}

	block, err := voiceChannels.SlackBlocksMultiChannel()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", block)

	voiceChannels.Muted(memberID1)
	if !voiceChannels.Channels[channelID1].Users[memberID1].Muted {
		t.Fatal("Expected the user is muted")
	}
	block, err = voiceChannels.SlackBlocksMultiChannel()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", block)

	exists = voiceChannels.Join(channel1, member1)
	if !exists {
		t.Fatal("Expected the user already joined the channel")
	}
	voiceChannels.Deafened(memberID1)
	if !voiceChannels.Channels[channelID1].Users[memberID1].Deafened {
		t.Fatal("Expected the user is deafened")
	}
	block, err = voiceChannels.SlackBlocksMultiChannel()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", block)

	voiceChannels.Join(channel1, member1)
	voiceChannels.Leave(memberID1)
	if len(voiceChannels.Channels[channelID1].Users) != 0 {
		t.Fatalf("Expected 0 user in voice channel, But %v",
			voiceChannels.Channels[channelID1].Users)
	}
	block, err = voiceChannels.SlackBlocksMultiChannel()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%+v", block)
}

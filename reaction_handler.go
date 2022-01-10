package main

type ReactionHandler interface {
	GetReaction(channel string, timestamp string) error
	GetEmojiURI(name string) string

	AddEmoji(name, value string)
	RemoveEmoji(name string)
}

package main

type MessageEscaper interface {
	EscapeMessage(content string) (output string, err error)
}

type MessageGetter interface {
	GetMessage(channelID, timestamp string) (string, error)
}

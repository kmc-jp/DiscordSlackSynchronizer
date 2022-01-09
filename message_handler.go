package main

type MessageEscaper interface {
	EscapeMessage(content string) (output string, err error)
}

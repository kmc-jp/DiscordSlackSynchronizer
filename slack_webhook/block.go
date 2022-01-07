package slack_webhook

import (
	"encoding/json"
	"fmt"
)

type BlockBase struct {
	Type     string         `json:"type"`
	Text     BlockElement   `json:"text,omitempty"`
	Elements []BlockElement `json:"elements,omitempty"`
	ImageURL string         `json:"image_url"`
	AltText  string         `json:"alt_text"`
	Title    BlockTitle     `json:"title"`
}

type BlockElement struct {
	Type     string `json:"type,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	AltText  string `json:"alt_text,omitempty"`
	Text     string `json:"text,omitempty"`
}

type BlockTitle struct {
	Type  string `json:"type,omitempty"`
	Text  string `json:"text,omitempty"`
	Emoji bool   `json:"emoji,omitempty"`
}

func ImageBlock(imageURL, altText string) BlockBase {
	return BlockBase{
		Type:     "image",
		AltText:  altText,
		ImageURL: imageURL,
	}
}

func ImageTitle(title string, emoji bool) BlockTitle {
	return BlockTitle{
		Type:  "plain_text",
		Emoji: emoji,
		Text:  title,
	}
}

func MrkdwnElement(text string) BlockElement {
	return BlockElement{Type: "mrkdwn", Text: text}
}

func ImageElement(imageURL, altText string) BlockElement {
	return BlockElement{
		Type:     "image",
		ImageURL: imageURL,
		AltText:  altText,
	}
}

func ContextBlock(elements ...BlockElement) BlockBase {
	return BlockBase{
		Type:     "context",
		Elements: elements,
	}
}

func DividerBlock() BlockBase {
	return BlockBase{Type: "divider"}
}

func SectionBlock() BlockBase {
	return BlockBase{Type: "section"}
}

func (b BlockBase) MarshalJSON() ([]byte, error) {
	switch b.Type {
	case "image":
		if b.Title.Type != "" {
			// image has title object
			type baseImage struct {
				Type     string     `json:"type"`
				ImageURL string     `json:"image_url"`
				AltText  string     `json:"alt_text"`
				Title    BlockTitle `json:"title"`
			}
			return json.Marshal(baseImage{b.Type, b.ImageURL, b.AltText, b.Title})
		}

		// no title object
		type baseImage struct {
			Type     string `json:"type"`
			ImageURL string `json:"image_url"`
			AltText  string `json:"alt_text"`
		}
		return json.Marshal(baseImage{b.Type, b.ImageURL, b.AltText})
	case "divider":
		type base struct {
			Type string `json:"type"`
		}
		return json.Marshal(base{"divider"})
	case "context":
		type baseElem struct {
			Type     string         `json:"type"`
			Elements []BlockElement `json:"elements,omitempty"`
		}
		return json.Marshal(baseElem{b.Type, b.Elements})
	}

	if b.Text.Type != "" {
		type baseText struct {
			Type string       `json:"type"`
			Text BlockElement `json:"text,omitempty"`
		}
		return json.Marshal(baseText{b.Type, b.Text})
	}

	return nil, fmt.Errorf("IlligalFormat")
}

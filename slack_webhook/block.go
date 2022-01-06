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

func ImageBlock(url, altText string) BlockBase {
	return BlockBase{
		Type:     "image",
		AltText:  altText,
		ImageURL: url,
	}
}

func ImageTitle(title string, emoji bool) BlockTitle {
	return BlockTitle{
		Type:  "plain_text",
		Emoji: emoji,
		Text:  title,
	}
}

func (b BlockBase) MarshalJSON() ([]byte, error) {
	if b.Type == "image" {
		if b.Title.Type != "" {
			type baseImage struct {
				Type     string     `json:"type"`
				ImageURL string     `json:"image_url"`
				AltText  string     `json:"alt_text"`
				Title    BlockTitle `json:"title"`
			}
			return json.Marshal(baseImage{b.Type, b.ImageURL, b.AltText, b.Title})
		}
		type baseImage struct {
			Type     string `json:"type"`
			ImageURL string `json:"image_url"`
			AltText  string `json:"alt_text"`
		}
		return json.Marshal(baseImage{b.Type, b.ImageURL, b.AltText})
	}
	if b.Text.Type != "" {
		type baseText struct {
			Type string       `json:"type"`
			Text BlockElement `json:"text,omitempty"`
		}
		return json.Marshal(baseText{b.Type, b.Text})
	}
	if len(b.Elements) > 0 {
		type baseElem struct {
			Type     string         `json:"type"`
			Elements []BlockElement `json:"elements,omitempty"`
		}
		return json.Marshal(baseElem{b.Type, b.Elements})
	}

	if b.Type != "" {
		type base struct {
			Type string `json:"type"`
		}
		return json.Marshal(base{b.Type})
	}

	return nil, fmt.Errorf("IlligalFormat")
}

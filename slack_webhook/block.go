package slack_webhook

import (
	"encoding/json"
	"fmt"
)

type BlockBase struct {
	Type     string         `json:"type"`
	Text     BlockElement   `json:"text,omitempty"`
	Elements []BlockElement `json:"elements,omitempty"`
}

type BlockElement struct {
	Type     string `json:"type,omitempty"`
	ImageURL string `json:"image_url,omitempty"`
	AltText  string `json:"alt_text,omitempty"`
	Text     string `json:"text,omitempty"`
}

func (b BlockBase) MarshalJSON() ([]byte, error) {
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

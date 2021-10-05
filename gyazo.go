package main

import (
	"bytes"

	"github.com/tomohiro/go-gyazo/gyazo"
)

type GyazoHandler struct {
	*gyazo.Client
	token string
}

func NewGyazoHandler(token string) (*GyazoHandler, error) {
	gyazo, err := gyazo.NewClient(Tokens.Gyazo.API)
	if err != nil {
		return nil, err
	}

	return &GyazoHandler{gyazo, token}, nil
}

//GyazoFileUpload Upload image to Gyazo and return PermalinkURL
func (g *GyazoHandler) UploadFromBytes(dataBytes []byte) (*gyazo.Image, error) {
	return g.Upload(bytes.NewReader(dataBytes))
}

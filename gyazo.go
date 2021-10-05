package main

import (
	"bytes"

	"github.com/tomohiro/go-gyazo/gyazo"
)

//GyazoFileUpload Upload image to Gyazo and return PermalinkURL
func GyazoFileUpload(dataBytes []byte) (string, error) {
	gyazo, err := gyazo.NewClient(Tokens.Gyazo.API)
	if err != nil {
		return "", err
	}

	image, err := gyazo.Upload(bytes.NewReader(dataBytes))
	if err != nil {
		return "", err
	}
	return image.PermalinkURL, nil
}

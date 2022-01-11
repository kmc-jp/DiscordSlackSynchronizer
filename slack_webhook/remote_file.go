package slack_webhook

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"

	"github.com/pkg/errors"
	"github.com/slack-go/slack"
)

func (s *Handler) FilesRemoteInfo(externalID, fileID string) (*slack.File, error) {
	return s.filesRemoteRequest("info", externalID, fileID)
}

func (s *Handler) FilesRemoteRemove(externalID, fileID string) error {
	_, err := s.filesRemoteRequest("remove", externalID, fileID)
	return err
}

func (s *Handler) filesRemoteRequest(method, externalID, fileID string) (*slack.File, error) {
	var req *http.Request

	var value = make(url.Values)

	if externalID != "" {
		value.Set("external_id", externalID)
	}

	if fileID != "" {
		value.Set("file", fileID)
	}

	var reqURI string
	switch method {
	case "info":
		reqURI = fmt.Sprintf("%s/files.remote.info?%s", SlackAPIEndpoint, value.Encode())
	case "remove":
		reqURI = fmt.Sprintf("%s/files.remote.remove?%s", SlackAPIEndpoint, value.Encode())
	}

	req, err := http.NewRequest("GET", reqURI, nil)
	if err != nil {
		return nil, errors.Wrap(err, "Requrst")
	}

	req.Header.Set("Authorization", "Bearer "+s.token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Sending")
	}
	defer resp.Body.Close()

	var responseAttr struct {
		OK    bool       `json:"ok"`
		Error string     `json:"error"`
		File  slack.File `json:"file"`
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return &responseAttr.File, errors.Wrap(err, "ReadAll")
	}

	err = json.Unmarshal(body, &responseAttr)
	if err != nil {
		return &responseAttr.File, errors.Wrapf(err, "DecodingJSON: %s", body)
	}

	if !responseAttr.OK {
		return &responseAttr.File, errors.New("SlackAPIError: " + responseAttr.Error)
	}

	return &responseAttr.File, nil
}

func (s *Handler) FilesRemoteAdd(file FilesRemoteAddParameters) (*slack.File, error) {
	var body = new(bytes.Buffer)

	var mw = multipart.NewWriter(body)

	pw, err := mw.CreateFormField("external_id")
	if err != nil {
		return nil, errors.Wrap(err, "CreatingPartAtExternalID")
	}

	pw.Write([]byte(file.ExternalID))

	pw, err = mw.CreateFormField("external_url")
	if err != nil {
		return nil, errors.Wrap(err, "CreatingPartAtExternalURL")
	}

	pw.Write([]byte(file.ExternalURL))

	pw, err = mw.CreateFormField("title")
	if err != nil {
		return nil, errors.Wrap(err, "CreatingPartAtInitialComment")
	}

	pw.Write([]byte(file.Title))

	if file.FileType != "" {
		pw, err := mw.CreateFormField("filetype")
		if err != nil {
			return nil, errors.Wrap(err, "CreatingPartAtFileType")
		}

		pw.Write([]byte(file.FileType))
	}

	if file.IndexableFileContents != nil {
		pw, err := mw.CreateFormField("indexable_file_contents")
		if err != nil {
			return nil, errors.Wrap(err, "CreatingPartAtIndexableFileContents")
		}

		io.Copy(pw, file.IndexableFileContents)
	}

	if file.PreviewImage != nil {
		pw, err := mw.CreateFormField("preview_image")
		if err != nil {
			return nil, errors.Wrap(err, "CreatingPartAtPreviewImage")
		}

		io.Copy(pw, file.PreviewImage)
	}
	mw.Close()

	var req *http.Request
	var reqURI = fmt.Sprintf("%s/files.remote.add", SlackAPIEndpoint)

	req, err = http.NewRequest("POST", reqURI, body)
	if err != nil {
		return nil, errors.Wrap(err, "Requrst")
	}

	req.Header.Set("Authorization", "Bearer "+s.token)
	req.Header.Set("Content-Type", mw.FormDataContentType())

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Sending")
	}
	defer resp.Body.Close()

	var responseAttr struct {
		OK    bool       `json:"ok"`
		Error string     `json:"error"`
		File  slack.File `json:"file"`
	}

	err = func() error {
		defer func() {
			err := recover()
			if err != nil {
				log.Printf("Panic in Decoding JSON: %v\n", err)
			}
		}()
		return json.NewDecoder(resp.Body).Decode(&responseAttr)
	}()

	if err != nil {
		return &responseAttr.File, errors.Wrap(err, "DecodingJSON")
	}

	if !responseAttr.OK {
		return &responseAttr.File, errors.New("SlackAPIError: " + responseAttr.Error)
	}

	return &responseAttr.File, nil
}

package slack_alerts

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
)

// Sends a simple message to a webhook
func SendTextMessage(url string, text string) error {
	return SendMessage(url, &Message{Text: text})
}

func SendMessage(url string, msg *Message) error {
	buf, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	reader := bytes.NewReader(buf)
	resp, err := http.Post(url, "application/json", reader)
	if err != nil {
		return err
	}

	if resp.StatusCode != 200 {
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		bodyString := string(bodyBytes)
		return &MessageError{StatusCode: resp.StatusCode, Err: bodyString}
	}

	return nil
}

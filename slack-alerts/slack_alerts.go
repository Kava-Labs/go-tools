package slack_alerts

import (
	"fmt"

	"github.com/slack-go/slack"
)

// SendMessage sends a one off message to a given slack channel
func SendMessage(token string, channel_id string, options ...slack.MsgOption) error {
	api := slack.New(token)

	channelID, timestamp, err := api.PostMessage(
		channel_id,
		options...,
	)

	if err != nil {
		return err
	}
	fmt.Printf("Message successfully sent to channel %s at %s", channelID, timestamp)

	return nil
}

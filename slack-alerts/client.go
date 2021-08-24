package slack_alerts

import (
	"fmt"

	"github.com/slack-go/slack"
)

// SlackAlerter is reusable client with helpful methods for a simple way to send messages
type SlackAlerter struct {
	SlackClient slack.Client
}

// NewSlackAlerter returns a new SlackAlerter with a given bot token
func NewSlackAlerter(slackToken string) SlackAlerter {
	return SlackAlerter{*slack.New(slackToken)}
}

// SendMessage sends a string message to a given channel
func (s *SlackAlerter) SendMessage(channelId string, text string) error {
	channelId, _, err := s.SlackClient.PostMessage(
		channelId,
		slack.MsgOptionText(text, true),
	)

	return err
}

// Info sends an INFO level message to a given channel
func (s *SlackAlerter) Info(channelId string, text string) error {
	return s.SendMessage(channelId, fmt.Sprintf("*`[INFO]`* %s", text))
}

// Warn sends an WARN level message to a given channel
func (s *SlackAlerter) Warn(channelId string, text string) error {
	return s.SendMessage(channelId, fmt.Sprintf(":warning: *`[WARN]`* %s", text))
}

// Error sends an ERROR level message to a given channel
func (s *SlackAlerter) Error(channelId string, text string) error {
	return s.SendMessage(channelId, fmt.Sprintf(":alert: *`[ERROR]`* %s", text))
}

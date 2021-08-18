package slack_alerts

import (
	"fmt"

	"github.com/slack-go/slack"
)

// A reusable client with helpful methods for a simple way to send messages
type SlackAlerter struct {
	SlackClient slack.Client
}

func NewSlackAlerter(slackToken string) SlackAlerter {
	return SlackAlerter{*slack.New(slackToken)}
}

func (s *SlackAlerter) SendMessage(channelId string, text string) error {
	channelId, _, err := s.SlackClient.PostMessage(
		channelId,
		slack.MsgOptionText(text, true),
	)

	return err
}

func (s *SlackAlerter) Info(channelId string, text string) error {
	return s.SendMessage(channelId, fmt.Sprintf("*`[INFO]`* %s", text))
}

func (s *SlackAlerter) Warn(channelId string, text string) error {
	return s.SendMessage(channelId, fmt.Sprintf(":warning: *`[WARN]`* %s", text))
}

func (s *SlackAlerter) Error(channelId string, text string) error {
	return s.SendMessage(channelId, fmt.Sprintf(":alert: *`[ERROR]`* %s", text))
}

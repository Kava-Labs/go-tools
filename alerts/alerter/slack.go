package alerter

import (
	"fmt"

	"github.com/slack-go/slack"
)

// SlackAlerter is reusable client with helpful methods for a simple way to send messages
type SlackAlerter struct {
	webhookUrl string
}

// Verify interface compliance at compile time
var _ Alerter = (*SlackAlerter)(nil)

// NewSlackAlerter returns a new SlackAlerter that posts messages to a specific webhook
func NewSlackAlerter(webhookUrl string) SlackAlerter {
	return SlackAlerter{webhookUrl: webhookUrl}
}

// SendMessage sends a string message to a given channel
func (s *SlackAlerter) SendMessage(message string) error {
	return slack.PostWebhook(s.webhookUrl, &slack.WebhookMessage{Text: message})
}

// Info sends an INFO level message to a given channel
func (s *SlackAlerter) Info(text string) error {
	return s.SendMessage(fmt.Sprintf("[INFO] %s", text))
}

// Warn sends an WARN level message to a given channel
func (s *SlackAlerter) Warn(text string) error {
	return s.SendMessage(fmt.Sprintf(":warning: [WARN] %s", text))
}

// Error sends an ERROR level message to a given channel
func (s *SlackAlerter) Error(text string) error {
	return s.SendMessage(fmt.Sprintf(":alert: [ERROR] %s", text))
}

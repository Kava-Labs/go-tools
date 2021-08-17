package slack_alerts

import (
	"github.com/slack-go/slack"
)

// Sends a simple message to a webhook
func SendTextMessage(url string, text string) error {
	return SendMessage(url, &slack.WebhookMessage{Text: text})
}

func SendMessage(url string, msg *slack.WebhookMessage) error {
	return slack.PostWebhook(url, msg)
}

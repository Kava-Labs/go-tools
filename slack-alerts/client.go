package slack_alerts

import "github.com/slack-go/slack"

// A reusable client to send messages so that you do not need to keep passing a webhook url around
type Client struct {
	webhookUrl string
}

func NewClient(webhookUrl string) Client {
	return Client{webhookUrl}
}

func (c *Client) SendMessage(msg *slack.WebhookMessage) error {
	return SendMessage(c.webhookUrl, msg)
}

func (c *Client) SendTextMessage(text string) error {
	return c.SendMessage(&slack.WebhookMessage{Text: text})
}

package slack_alerts

// A reusable client to send messages so that you do not need to keep passing a webhook url around
type Client struct {
	webhookUrl string
}

func NewClient(webhookUrl string) Client {
	return Client{webhookUrl}
}

func (c *Client) SendMessage(msg *Message) error {
	return SendMessage(c.webhookUrl, msg)
}

func (c *Client) SendTextMessage(text string) error {
	return c.SendMessage(&Message{Text: text})
}

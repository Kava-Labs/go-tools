package slack_alerts

import "github.com/slack-go/slack"

// A reusable client with helpful methods for a simple way to send messages
type SlackAlerter struct {
	SlackClient slack.Client
}

func NewClient(slackToken string) SlackAlerter {
	return SlackAlerter{*slack.New(slackToken)}
}

func (s *SlackAlerter) Info(channelId string, title string, content string) error {
	headerText := slack.NewTextBlockObject("plain_text", title, true, false)
	headerBlock := slack.NewHeaderBlock(headerText)

	divSection := slack.NewDividerBlock()

	contentText := slack.NewTextBlockObject("plain_text", content, true, false)
	contentSection := slack.NewSectionBlock(contentText, nil, nil)

	channelId, _, err := s.SlackClient.PostMessage(
		channelId,
		slack.MsgOptionBlocks(headerBlock, divSection, contentSection),
	)

	return err
}

package slack_alerts

import "fmt"

type MessageError struct {
	// HTTP status code returned from the request
	StatusCode int
	// Error message from Slack. Common errors can be found here:
	// https://api.slack.com/messaging/webhooks#handling_errors
	Err string
}

func (e *MessageError) Error() string {
	return fmt.Sprint(e.StatusCode, e.Err)
}

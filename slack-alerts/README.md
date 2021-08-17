# Slack Alerts

Package for easily sending messages to a Slack channel.

## Slack Setup

1. [Create a Slack app](https://api.slack.com/apps/new)
2. Activate incoming webhooks and add a new webhook to your workspace
3. Copy your webhook URL

## Basic Example

```golang
package main

import (
	"log"
	"os"

	slack_alerts "github.com/kava-labs/go-tools/slack-alerts"
)

func main() {
	webhook_url := os.Getenv("WEBHOOK_URL")
	err := slack_alerts.SendTextMessage(webhook_url, "Hello World")

	if err != nil {
		log.Print("Error sending message", err)
	}
}
```

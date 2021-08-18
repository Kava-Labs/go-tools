package main

import (
	"log"
	"os"

	slack_alerts "github.com/kava-labs/go-tools/slack-alerts"
)

func main() {
	slackToken := os.Getenv("SLACK_TOKEN")
	channelId := os.Getenv("SLACK_CHANNEL")
	alertClient := slack_alerts.NewSlackAlerter(slackToken)

	if err := alertClient.Info(channelId, "something happened"); err != nil {
		log.Print("Failed to send message")
	}

	if err := alertClient.Warn(channelId, "watch out"); err != nil {
		log.Print("Failed to send message")
	}

	if err := alertClient.Error(channelId, "oh no"); err != nil {
		log.Print("Failed to send message")
	}
}

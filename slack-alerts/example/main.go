package main

import (
	"log"
	"os"

	slack_alerts "github.com/kava-labs/go-tools/slack-alerts"
)

func main() {
	slackToken := os.Getenv("SLACK_TOKEN")
	alertClient := slack_alerts.NewClient(slackToken)

	err := alertClient.Info("C02AWS3BBGX", "Some title", "something happened")

	if err != nil {
		log.Print("Error sending message ", err)
	}
}

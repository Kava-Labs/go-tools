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

package main

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

func main() {
	// Load config
	config, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	c := *config

	ctx := context.Background()
	claimQueue := make(chan server.ClaimJob, claimer.JobQueueSize)

	dispatcher := claimer.NewDispatcher(claimQueue)
	go dispatcher.Start(ctx, c)

	s := server.NewServer(claimQueue)
	log.Info("Starting server...")
	go log.Fatal(s.Start())

}

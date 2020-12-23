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
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	dispatcher := claimer.NewDispatcher()
	ctx := context.Background()
	go dispatcher.Start(ctx, cfg)

	s := server.NewServer(dispatcher.JobQueue())
	log.Info("Starting server...")
	go func() {
		if err := s.Start(); err != nil {
			log.Fatal(err)
		}
	}()

}

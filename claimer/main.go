package main

import (
	"context"

	"github.com/kava-labs/kava/app"

	log "github.com/sirupsen/logrus"

	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

func main() {
	cfg, err := config.LoadConfigFromEnvs("CLAIMER_")
	if err != nil {
		log.Fatal(err)
	}

	app.SetSDKConfig()

	log.SetLevel(log.DebugLevel)

	dispatcher := claimer.NewDispatcher(cfg)
	ctx := context.Background()
	go dispatcher.Start(ctx)

	s := server.NewServer(dispatcher)
	log.Info("Starting server...")
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}

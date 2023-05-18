package main

import (
	"context"

	"github.com/kava-labs/kava/app"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

func main() {
	configPath := pflag.String("config", config.DefaultConfigPath, "path to config file")
	pflag.Parse()

	app.SetSDKConfig()

	// Load config
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(log.DebugLevel)

	dispatcher := claimer.NewDispatcher(cfg)
	ctx := context.Background()
	go dispatcher.Start(ctx)

	s := server.NewServer(dispatcher.JobQueue())
	log.Info("Starting server...")
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	kava "github.com/kava-labs/kava/app"
	"github.com/sirupsen/logrus"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/pflag"

	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

func main() {
	configPath := pflag.String("config", config.DefaultConfigPath, "path to config file")
	pflag.Parse()

	// Load kava claimers
	sdkConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(sdkConfig)

	// Load config
	cfg, err := config.GetConfig(*configPath)
	if err != nil {
		panic(err)
	}

	log.SetLevel(logrus.DebugLevel)

	dispatcher := claimer.NewDispatcher(*cfg)
	ctx := context.Background()
	go dispatcher.Start(ctx)

	s := server.NewServer(dispatcher.JobQueue())
	log.Info("Starting server...")
	if err := s.Start(); err != nil {
		log.Fatal(err)
	}
}

package main

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	kava "github.com/kava-labs/kava/app"
	log "github.com/sirupsen/logrus"

	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/server"
)

func main() {
	// Load kava claimers
	sdkConfig := sdk.GetConfig()
	kava.SetBech32AddressPrefixes(sdkConfig)

	// Load config
	cfg, err := config.GetConfig()
	if err != nil {
		panic(err)
	}

	dispatcher := claimer.NewDispatcher(*cfg)
	ctx := context.Background()
	go dispatcher.Start(ctx)

	s := server.NewServer(dispatcher.JobQueue())
	log.Info("Starting server...")
	go func() {
		if err := s.Start(); err != nil {
			log.Fatal(err)
		}
	}()

}

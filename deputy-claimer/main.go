package main

import (
	"context"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	"github.com/rs/zerolog"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
	"github.com/kava-labs/go-tools/deputy-claimer/config"
	"github.com/kava-labs/go-tools/deputy-claimer/health"
)

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

	// Set global address prefixes
	kavaConfig := sdk.GetConfig()
	app.SetBech32AddressPrefixes(kavaConfig)
	kavaConfig.Seal()

	cfg, err := config.LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("could not load config")
	}

	kavaClaimer := claim.NewKavaClaimer(
		cfg.KavaGrpcURL,
		cfg.BnbRPCURL,
		cfg.Deputies,
		cfg.KavaMnemonics,
	)
	bnbClaimer := claim.NewBnbClaimer(
		cfg.KavaGrpcURL,
		cfg.BnbRPCURL,
		cfg.Deputies,
		cfg.BnbMnemonics,
	)

	ctx := context.Background()

	health.StartHealthCheckService(
		ctx,
		logger,
		kavaClaimer,
	)

	kavaClaimer.Start(ctx)
	bnbClaimer.Start(ctx)

	select {}
}

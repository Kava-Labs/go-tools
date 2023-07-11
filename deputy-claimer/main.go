package main

import (
	"context"
	"os"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/kava/app"
	"github.com/rs/zerolog"
	"github.com/spf13/viper"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
)

type Config struct {
	BnbRPCURL     string
	KavaGrpcURL   string
	Deputies      claim.DeputyAddresses
	BnbMnemonics  []string
	KavaMnemonics []string
}

type ConfigSimple struct {
	BnbRPCURL   string
	KavaGrpcURL string
	Deputies    map[string]struct {
		Kava string
		Bnb  string
	}
	BnbMnemonics  []string
	KavaMnemonics []string
}

func loadConfig() (Config, error) {
	v := viper.New()
	v.SetConfigName("config") // name of config file (without extension)
	v.SetConfigType("toml")
	v.AddConfigPath("$HOME")
	if err := v.ReadInConfig(); err != nil {
		return Config{}, err
	}

	var temp ConfigSimple
	if err := v.Unmarshal(&temp); err != nil {
		return Config{}, err
	}

	deputies := claim.DeputyAddresses{}
	for k, v := range temp.Deputies {
		deputies[k] = claim.DeputyAddress{
			Kava: mustDecodeKavaAddress(v.Kava),
			Bnb:  mustDecodeBnbAddress(v.Bnb),
		}
	}

	cfg := Config{
		BnbRPCURL:     temp.BnbRPCURL,
		KavaGrpcURL:   temp.KavaGrpcURL,
		Deputies:      deputies,
		BnbMnemonics:  temp.BnbMnemonics,
		KavaMnemonics: temp.KavaMnemonics,
	}
	return cfg, nil
}

func main() {
	logger := zerolog.New(os.Stdout).With().Timestamp().Caller().Logger()

	// Set global address prefixes
	kavaConfig := sdk.GetConfig()
	app.SetBech32AddressPrefixes(kavaConfig)
	kavaConfig.Seal()

	cfg, err := loadConfig()
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
	startHealthCheckService(
		ctx,
		logger,
		cfg,
		kavaClaimer,
	)

	kavaClaimer.Start(ctx)
	bnbClaimer.Start(ctx)

	select {}
}

func mustDecodeKavaAddress(address string) sdk.AccAddress {
	aa, err := sdk.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return aa
}

func mustDecodeBnbAddress(address string) bnbtypes.AccAddress {
	aa, err := bnbtypes.AccAddressFromBech32(address)
	if err != nil {
		panic(err)
	}
	return aa
}

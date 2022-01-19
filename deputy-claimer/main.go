package main

import (
	"context"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/common/types"
	"github.com/kava-labs/kava/app"
	"github.com/spf13/viper"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
)

type Config struct {
	BnbRPCURL     string
	kavaGrpcURL   string
	Deputies      claim.DeputyAddresses
	BnbMnemonics  []string
	KavaMnemonics []string
}

type ConfigSimple struct {
	BnbRPCURL   string
	kavaGrpcURL string
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
		kavaGrpcURL:   temp.kavaGrpcURL,
		Deputies:      deputies,
		BnbMnemonics:  temp.BnbMnemonics,
		KavaMnemonics: temp.KavaMnemonics,
	}
	return cfg, nil
}

func main() {

	// Set global address prefixes
	kavaConfig := sdk.GetConfig()
	app.SetBech32AddressPrefixes(kavaConfig)
	kavaConfig.Seal()

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	kavaClaimer := claim.NewKavaClaimer(
		cfg.kavaGrpcURL,
		cfg.BnbRPCURL,
		cfg.Deputies,
		cfg.KavaMnemonics,
	)
	bnbClaimer := claim.NewBnbClaimer(
		cfg.kavaGrpcURL,
		cfg.BnbRPCURL,
		cfg.Deputies,
		cfg.BnbMnemonics,
	)

	ctx := context.Background()
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

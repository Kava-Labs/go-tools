package main

import (
	"context"
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/kava/app"
	"github.com/spf13/viper"

	"github.com/kava-labs/go-tools/deputy-claimer/claim"
)

type Config struct {
	BnbRPCURL     string
	KavaRestURL   string
	KavaRPCURL    string
	Deputies      claim.DeputyAddresses
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

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func main() {

	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Set global address prefixes
	kavaConfig := sdk.GetConfig()
	app.SetBech32AddressPrefixes(kavaConfig) // XXX G34 descend only one level of abstraction
	kavaConfig.Seal()

	// XXX G30 functions should do one thing

	// XXX F1 too many arguments
	// XXX G5 duplication
	kavaClaimer := claim.NewKavaClaimer(cfg.KavaRestURL, cfg.KavaRPCURL, cfg.BnbRPCURL, cfg.Deputies, cfg.KavaMnemonics)
	bnbClaimer := claim.NewBnbClaimer(cfg.KavaRestURL, cfg.KavaRPCURL, cfg.BnbRPCURL, cfg.Deputies, cfg.BnbMnemonics)

	ctx := context.Background() // XXX G34 too many levels of abstraction
	kavaClaimer.Run(ctx)
	bnbClaimer.Run(ctx) // XXX G5 duplication

	select {}
}

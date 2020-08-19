package main

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	kavaapp "github.com/kava-labs/kava/app"
	"github.com/spf13/viper"

	"github.com/kava-labs/go-tools/sentinel/app"
)

type Config struct {
	RestURL          string
	CdpOwnerMnemonic string
	CdpDenom         string
	ChainID          string
	LowerTrigger     string
	UpperTrigger     string
}

func loadConfig() (Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.AddConfigPath("$HOME/.sentinel")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return Config{}, err
	}
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return cfg, err
	}
	return cfg, nil
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	config := sdk.GetConfig()
	kavaapp.SetBech32AddressPrefixes(config)
	kavaapp.SetBip44CoinType(config)
	config.Seal()

	lowerTrigger, err := sdk.NewDecFromStr(cfg.LowerTrigger)
	if err != nil {
		log.Fatal(err)
	}
	upperTrigger, err := sdk.NewDecFromStr(cfg.UpperTrigger)
	if err != nil {
		log.Fatal(err)
	}
	app, err := app.NewDefaultApp(cfg.RestURL, cfg.CdpOwnerMnemonic, cfg.CdpDenom, cfg.ChainID, lowerTrigger, upperTrigger)
	if err != nil {
		log.Fatal(err)
	}
	log.Println("starting app")
	app.Run()
}

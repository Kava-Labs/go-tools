package main

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/spf13/viper"

	"github.com/kava-labs/go-tools/sentinel/app"
)

type Config struct {
	RestURL          string
	CdpOwnerMnemonic string
	CdpDenom         string
	ChainID          string
	LowerTrigger     sdk.Dec
	UpperTrigger     sdk.Dec
}

func main() {
	viper.SetConfigName("config")
	viper.AddConfigPath("$HOME/.sentinel")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		log.Fatal(err)
	}
	var cfg Config
	err = viper.Unmarshal(&cfg)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("starting app")
	app := app.NewDefaultApp(cfg.RestURL, cfg.CdpOwnerMnemonic, cfg.CdpDenom, cfg.ChainID, cfg.LowerTrigger, cfg.UpperTrigger)
	app.Run()
}

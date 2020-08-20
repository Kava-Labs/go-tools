package main

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	kavaapp "github.com/kava-labs/kava/app"
	"github.com/spf13/viper"

	"github.com/kava-labs/go-tools/sentinel/app"
)

type Config struct {
	// RestURL the address of a kava node rest server
	RestURL string
	// CdpOwnerMnemonic is the mnemonic for an address of a CDP
	CdpOwnerMnemonic string
	// CdpDenom is the collateral type for the CDP, eg bnb
	CdpDenom string
	// LowerTrigger is collateral ratio under which the bot will repay debt
	LowerTrigger string
	// UpperTrigger is the collatereal ratio above which the bot will draw more debt
	UpperTrigger string
	ChainID      string
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

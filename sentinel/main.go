package main

import (
	"log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/sentinel/app"
)

func main() {
	// TODO add to a config
	restURL := "http://kava3.data.kava.io"
	cdpOwnerMnemonic := "" // TODO
	cdpDenom := "bnb"
	chainID := "kava-3"
	lowerTrigger := sdk.MustNewDecFromStr("2.00")
	upperTrigger := sdk.MustNewDecFromStr("2.50")

	log.Println("starting app")
	app.NewApp(restURL, cdpOwnerMnemonic, cdpDenom, chainID, lowerTrigger, upperTrigger).Run()
}

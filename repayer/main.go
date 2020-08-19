package main

import (
	"log"

	"github.com/kava-labs/go-tools/repayer/app"
)

func main() {
	// TODO add to a config
	restURL := "http://kava3.data.kava.io"
	cdpOwnerMnemonic := "" // TODO
	cdpDenom := "bnb"
	chainID := "kava-3"

	log.Println("starting app")
	app.NewApp(restURL, cdpOwnerMnemonic, cdpDenom, chainID).Run()
}

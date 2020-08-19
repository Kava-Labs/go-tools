package main

import (
	"log"

	"github.com/kava-labs/go-tools/repayer/app"
)

func main() {
	restURL := "http://kava3.data.kava.io"
	cdpOwner := "kava12lsjquv3xrzyu27gyzuxtsmydk8akufznj8qsc"
	cdpDenom := "bnb"
	chainID := "kava-3"

	if err := app.NewApp(restURL, cdpOwner, cdpDenom, chainID).Run(); err != nil {
		log.Fatal(err)
	}
}

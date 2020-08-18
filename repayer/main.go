package main

import (
	"log"

	"github.com/kava-labs/go-tools/repayer/app"
)

func main() {
	restURL := "http://kava3.data.kava.io"
	cdpOwner := "kava12lsjquv3xrzyu27gyzuxtsmydk8akufznj8qsc"
	cdpDenom := "bnb"

	if err := app.NewApp(restURL, cdpOwner, cdpDenom).Run(); err != nil {
		log.Fatal(err)
	}
}

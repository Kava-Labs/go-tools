package main

import (
	"context"

	"github.com/kava-labs/go-tools/claimer/config"
	"github.com/kava-labs/go-tools/claimer/renamethis"
)

func main() {
	// Load config
	config, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	c := *config

	renamethis.Main(context.Background(), c)
}

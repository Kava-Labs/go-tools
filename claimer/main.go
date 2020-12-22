package main

import (
	"context"

	"github.com/kava-labs/go-tools/claimer/claimer"
	"github.com/kava-labs/go-tools/claimer/config"
)

func main() {
	// Load config
	config, err := config.GetConfig()
	if err != nil {
		panic(err)
	}
	c := *config

	claimer.Run(context.Background(), c)
}

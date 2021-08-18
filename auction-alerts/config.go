package main

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// ConfigLoader provides an interface for
// loading config values from a provided key
type ConfigLoader interface {
	Get(key string) string
}

// Config provides application configuration
type Config struct {
	KavaRpcUrl string
	// Interval at which the process runs to check ongoing auctions
	Interval       time.Duration
	SlackToken     string
	SlackChannelId string
	// US dollar value of auctions that triggers alert
	UsdThreshold sdk.Dec
}

// LoadConfig loads key values from a ConfigLoader
// and returns a new Config
func LoadConfig(loader ConfigLoader) (Config, error) {
	err := godotenv.Load()
	if err != nil {
		return Config{}, err
	}
	rpcURL := loader.Get(kavaRpcUrlEnvKey)
	if rpcURL == "" {
		return Config{}, fmt.Errorf("%s not set", kavaRpcUrlEnvKey)
	}

	slackToken := loader.Get(slackToken)
	slackChannelId := loader.Get(slackChannelId)
	usdThreshold := loader.Get(usdThreshold)

	usdThresholdDec, err := sdk.NewDecFromStr(usdThreshold)
	if err != nil {
		return Config{}, err
	}

	updateInterval, err := time.ParseDuration(loader.Get(interval))
	if err != nil {
		updateInterval = time.Duration(10 * time.Minute)
	}

	return Config{
		KavaRpcUrl:     rpcURL,
		Interval:       updateInterval,
		SlackToken:     slackToken,
		SlackChannelId: slackChannelId,
		UsdThreshold:   usdThresholdDec,
	}, nil
}

// EnvLoader loads keys from os environment
type EnvLoader struct {
}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

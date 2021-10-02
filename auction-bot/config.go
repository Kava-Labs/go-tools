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
	KavaRpcUrl         string
	KavaBidInterval    time.Duration
	KavaKeeperMnemonic string
	ProfitMargin       sdk.Dec
}

// LoadConfig loads key values from a ConfigLoader
// and returns a new Config
func LoadConfig(loader ConfigLoader) (Config, error) {
	err := godotenv.Load()
	if err != nil {
		fmt.Printf(".env not found, attempting to proceed with available env variables\n")
	}
	rpcURL := loader.Get(kavaRpcUrlEnvKey)
	if rpcURL == "" {
		return Config{}, fmt.Errorf("%s not set", kavaRpcUrlEnvKey)
	}

	keeperMnemonic := loader.Get(mnemonicEnvKey)

	marginStr := loader.Get(profitMargin)
	fmt.Printf("%s\n", marginStr)

	marginDec, err := sdk.NewDecFromStr(marginStr)
	if err != nil {
		return Config{}, err
	}

	keeperBidInterval, err := time.ParseDuration(loader.Get(bidInterval))
	if err != nil {
		keeperBidInterval = time.Duration(10 * time.Minute)
	}

	return Config{
		KavaRpcUrl:         rpcURL,
		KavaBidInterval:    keeperBidInterval,
		KavaKeeperMnemonic: keeperMnemonic,
		ProfitMargin:       marginDec,
	}, nil
}

// EnvLoader loads keys from os environment
type EnvLoader struct {
}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

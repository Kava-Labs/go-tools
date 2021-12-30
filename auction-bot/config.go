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
	KavaGrpcUrl        string
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
	grpcURL := loader.Get(kavaGrpcUrlEnvKey)
	if grpcURL == "" {
		return Config{}, fmt.Errorf("%s not set", kavaGrpcUrlEnvKey)
	}

	keeperMnemonic := loader.Get(mnemonicEnvKey)

	marginStr := loader.Get(profitMarginKey)
	if marginStr == "" {
		return Config{}, fmt.Errorf("%s not set", profitMarginKey)
	}

	marginDec, err := sdk.NewDecFromStr(marginStr)
	if err != nil {
		return Config{}, err
	}

	keeperBidInterval, err := time.ParseDuration(loader.Get(bidIntervalKey))
	if err != nil {
		keeperBidInterval = time.Duration(10 * time.Minute)
	}

	return Config{
		KavaGrpcUrl:        grpcURL,
		KavaBidInterval:    keeperBidInterval,
		KavaKeeperMnemonic: keeperMnemonic,
		ProfitMargin:       marginDec,
	}, nil
}

// EnvLoader loads keys from os environment
type EnvLoader struct{}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

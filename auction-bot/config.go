package main

import (
	"fmt"
	"os"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	kavaRpcUrlEnvKey        = "KAVA_RPC_URL"
	kavaBidIntervalEnvKey   = "KAVA_BID_INTERVAL"
	kavaKeeperAddressEnvKey = "KAVA_KEEPER_ADDRESS"
	profitMargin            = "BID_MARGIN"
)

// ConfigLoader provides an interface for
// loading config values from a provided key
type ConfigLoader interface {
	Get(key string) string
}

// Config provides application configuration
type Config struct {
	KavaRpcUrl        string
	KavaBidInterval   time.Duration
	KavaKeeperAddress sdk.AccAddress
	ProfitMargin      sdk.Dec
}

// LoadConfig loads key values from a ConfigLoader
// and returns a new Config
func LoadConfig(loader ConfigLoader) (Config, error) {
	rpcURL := loader.Get(kavaRpcUrlEnvKey)
	if rpcURL == "" {
		return Config{}, fmt.Errorf("%s not set", kavaRpcUrlEnvKey)
	}

	bidInterval, err := time.ParseDuration(loader.Get(kavaBidIntervalEnvKey))
	if err != nil {
		bidInterval = time.Duration(10 * time.Minute)
	}

	keeperBech32Address := loader.Get(kavaKeeperAddressEnvKey)
	if keeperBech32Address == "" {
		return Config{}, fmt.Errorf("%s not set", kavaKeeperAddressEnvKey)
	}

	keeperAddress, err := sdk.AccAddressFromBech32(keeperBech32Address)
	if err != nil {
		return Config{}, err
	}

	marginStr := loader.Get(profitMargin)

	marginDec, err := sdk.NewDecFromStr(marginStr)
	if err != nil {
		return Config{}, err
	}

	return Config{
		KavaRpcUrl:        rpcURL,
		KavaBidInterval:   bidInterval,
		KavaKeeperAddress: keeperAddress,
		ProfitMargin:      marginDec,
	}, nil
}

// EnvLoader loads keys from os environment
type EnvLoader struct {
}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

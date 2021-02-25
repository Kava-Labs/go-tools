package main

import (
	"fmt"
	"os"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	kavaRpcUrlEnvKey             = "KAVA_RPC_URL"
	kavaLiqudationIntervalEnvKey = "KAVA_LIQUIDATION_INTERVAL"
	kavaKeeperAddressEnvKey      = "KAVA_KEEPER_ADDRESS"
)

// ConfigLoader provides an interface for
// loading config values from a provided key
type ConfigLoader interface {
	Get(key string) string
}

// Config provides application configuration
type Config struct {
	KavaRpcUrl              string
	KavaLiquidationInterval time.Duration
	KavaKeeperAddress       sdk.AccAddress
}

// LoadConfig loads key values from a ConfigLoader
// and returns a new Config
func LoadConfig(loader ConfigLoader) (Config, error) {
	rpcUrl := loader.Get(kavaRpcUrlEnvKey)
	if rpcUrl == "" {
		return Config{}, fmt.Errorf("%s not set", kavaRpcUrlEnvKey)
	}

	liquidationInterval, err := time.ParseDuration(loader.Get(kavaLiqudationIntervalEnvKey))
	if err != nil {
		liquidationInterval = time.Duration(10 * time.Minute)
	}

	keeperBech32Address := loader.Get(kavaKeeperAddressEnvKey)
	if keeperBech32Address == "" {
		return Config{}, fmt.Errorf("%s not set", kavaKeeperAddressEnvKey)
	}

	keeperAddress, err := sdk.AccAddressFromBech32(keeperBech32Address)
	if err != nil {
		return Config{}, err
	}

	return Config{
		KavaRpcUrl:              loader.Get(kavaRpcUrlEnvKey),
		KavaLiquidationInterval: liquidationInterval,
		KavaKeeperAddress:       keeperAddress,
	}, nil
}

// EnvLoader loads keys from os environment
type EnvLoader struct {
}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

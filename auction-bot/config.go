package main

import (
	"fmt"
	"os"
	"time"

	"github.com/joho/godotenv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	kavaGrpcUrlEnvKey       = "KAVA_GRPC_URL"
	mnemonicEnvKey          = "KEEPER_MNEMONIC"
	profitMarginKey         = "BID_MARGIN"
	bidIntervalKey          = "BID_INTERVAL"
	heathCheckListenAddrKey = "HEALTH_CHECK_LISTEN_ADDR"
)

// ConfigLoader provides an interface for
// loading config values from a provided key
type ConfigLoader interface {
	Get(key string) string
}

// Config provides application configuration
type Config struct {
	KavaGrpcUrl          string
	KavaBidInterval      time.Duration
	KavaKeeperMnemonic   string
	ProfitMargin         sdk.Dec
	HeathCheckListenAddr string
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

	healthCheckListenAddr := loader.Get(heathCheckListenAddrKey)
	if healthCheckListenAddr == "" {
		healthCheckListenAddr = ":8080"
	}

	return Config{
		KavaGrpcUrl:          grpcURL,
		KavaBidInterval:      keeperBidInterval,
		KavaKeeperMnemonic:   keeperMnemonic,
		ProfitMargin:         marginDec,
		HeathCheckListenAddr: healthCheckListenAddr,
	}, nil
}

// EnvLoader loads keys from os environment
type EnvLoader struct{}

// Get retrieves key from environment
func (l *EnvLoader) Get(key string) string {
	return os.Getenv(key)
}

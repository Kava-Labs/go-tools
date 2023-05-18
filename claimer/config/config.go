package config

import (
	"fmt"
	"os"
	"strings"
)

// Config defines chain connections and mnemonics
type Config struct {
	Kava         KavaConfig
	BinanceChain BinanceChainConfig
}

// KavaConfig defines information required for Kava blockchain interaction
type KavaConfig struct {
	ChainID   string
	Endpoint  string
	Mnemonics []string
}

// BinanceChainConfig defines information required for Binance Chain interaction
type BinanceChainConfig struct {
	ChainID  string
	Endpoint string
	Mnemonic string
}

// LoadConfigFromEnvs reads env vars with the provided prefix and parses them into a Config
func LoadConfigFromEnvs(prefix string) (Config, error) {
	var config Config

	env, found := os.LookupEnv(prefix + "KAVA_CHAIN_ID")
	if !found {
		return Config{}, fmt.Errorf("env %sKAVA_CHAIN_ID is empty", prefix)
	}
	config.Kava.ChainID = env

	env, found = os.LookupEnv(prefix + "KAVA_ENDPOINT")
	if !found {
		return Config{}, fmt.Errorf("env %sKAVA_ENDPOINT is empty", prefix)
	}
	config.Kava.Endpoint = env

	env, found = os.LookupEnv(prefix + "KAVA_MNEMONICS")
	if !found {
		return Config{}, fmt.Errorf("env %sKAVA_MNEMONICS is empty", prefix)
	}
	config.Kava.Mnemonics = strings.Split(env, ",")

	env, found = os.LookupEnv(prefix + "BINANCE_CHAIN_ID")
	if !found {
		return Config{}, fmt.Errorf("env %sBINANCE_CHAIN_ID is empty", prefix)
	}
	config.BinanceChain.ChainID = env

	env, found = os.LookupEnv(prefix + "BINANCE_ENDPOINT")
	if !found {
		return Config{}, fmt.Errorf("env %sBINANCE_ENDPOINT is empty", prefix)
	}
	config.BinanceChain.Endpoint = env

	env, found = os.LookupEnv(prefix + "BINANCE_MNEMONIC")
	if !found {
		return Config{}, fmt.Errorf("env %sBINANCE_MNEMONIC is empty", prefix)
	}
	config.BinanceChain.Mnemonic = env

	return config, nil
}

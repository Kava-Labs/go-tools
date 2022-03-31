package config

import (
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
)

type AuctionsConfig struct {
	BaseConfig
	RpcConfig
	// US dollar value of auctions that triggers alert
	UsdThreshold                   sdk.Dec
	InefficientAuctionUSDThreshold sdk.Dec
	InefficientRatio               sdk.Dec
	InefficientTimeRemaining       time.Duration
}

// LoadAuctionsConfig loads key values from a ConfigLoader and returns a new AuctionsConfig
func LoadAuctionsConfig(loader ConfigLoader, logger log.Logger) (AuctionsConfig, error) {
	baseConfig, err := LoadBaseConfig(loader, logger)
	if err != nil {
		return AuctionsConfig{}, err
	}

	rpcConfig, err := LoadRpcConfig(loader)
	if err != nil {
		return AuctionsConfig{}, err
	}

	usdThreshold := loader.Get(usdThresholdEnvKey)

	usdThresholdDec, err := sdk.NewDecFromStr(usdThreshold)
	if err != nil {
		return AuctionsConfig{}, err
	}

	inefficientThreshold := loader.Get(inefficientThresholdEnvKey)
	inefficientThresholdDec, err := sdk.NewDecFromStr(inefficientThreshold)
	if err != nil {
		return AuctionsConfig{}, err
	}

	inefficientRatio := loader.Get(inefficientRatioEnvKey)
	inefficientRatioDec, err := sdk.NewDecFromStr(inefficientRatio)
	if err != nil {
		return AuctionsConfig{}, err
	}

	inefficientTimeRemaining := loader.Get(inefficientTimeRemainingEnvKey)
	inefficientTimeRemainingDuration, err := time.ParseDuration(inefficientTimeRemaining)
	if err != nil {
		return AuctionsConfig{}, err
	}

	return AuctionsConfig{
		BaseConfig:                     baseConfig,
		RpcConfig:                      rpcConfig,
		UsdThreshold:                   usdThresholdDec,
		InefficientAuctionUSDThreshold: inefficientThresholdDec,
		InefficientRatio:               inefficientRatioDec,
		InefficientTimeRemaining:       inefficientTimeRemainingDuration,
	}, nil
}

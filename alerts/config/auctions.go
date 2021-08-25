package config

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AuctionsConfig struct {
	BaseConfig
	// US dollar value of auctions that triggers alert
	UsdThreshold sdk.Dec
}

// LoadAuctionsConfig loads key values from a ConfigLoader and returns a new AuctionsConfig
func LoadAuctionsConfig(loader ConfigLoader) (AuctionsConfig, error) {
	baseConfig, err := LoadBaseConfig(loader)
	if err != nil {
		return AuctionsConfig{}, err
	}

	usdThreshold := loader.Get(usdThresholdEnvKey)

	usdThresholdDec, err := sdk.NewDecFromStr(usdThreshold)
	if err != nil {
		return AuctionsConfig{}, err
	}

	return AuctionsConfig{
		BaseConfig:   baseConfig,
		UsdThreshold: usdThresholdDec,
	}, nil
}

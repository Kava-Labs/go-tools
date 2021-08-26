package config

import (
	"fmt"
	"strconv"
)

const spreadPercentThresholdEnvKey = "SPREAD_PERCENT_THRESHOLD"

type ArbitrageConfig struct {
	BaseConfig
	// US dollar value of auctions that triggers alert
	SpreadPercentThreshold float64
}

// LoadAuctionsConfig loads key values from a ConfigLoader and returns a new AuctionsConfig
func LoadArbitrageConfig(loader ConfigLoader) (ArbitrageConfig, error) {
	baseConfig, err := LoadBaseConfig(loader)
	if err != nil {
		return ArbitrageConfig{}, err
	}

	spreadThresholdPercent := loader.Get(spreadPercentThresholdEnvKey)
	if spreadThresholdPercent == "" {
		return ArbitrageConfig{}, fmt.Errorf("%s not set", spreadPercentThresholdEnvKey)
	}

	spreadThreshold, err := strconv.ParseFloat(spreadThresholdPercent, 64)
	if err != nil {
		return ArbitrageConfig{}, err
	}

	return ArbitrageConfig{
		BaseConfig:             baseConfig,
		SpreadPercentThreshold: spreadThreshold,
	}, nil
}

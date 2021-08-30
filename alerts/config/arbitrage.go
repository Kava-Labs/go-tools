package config

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const spreadPercentThresholdEnvKey = "SPREAD_PERCENT_THRESHOLD"

// ArbitrageConfig defines the swap arbitrage specific configuration fields
type ArbitrageConfig struct {
	Base BaseConfig
	// Spread percent that triggers alert
	SpreadPercentThreshold sdk.Dec
}

// LoadArbitrageConfig loads key values from a ConfigLoader and returns a new ArbitrageConfig
func LoadArbitrageConfig(loader ConfigLoader) (ArbitrageConfig, error) {
	baseConfig, err := LoadBaseConfig(loader)
	if err != nil {
		return ArbitrageConfig{}, err
	}

	spreadThresholdPercent := loader.Get(spreadPercentThresholdEnvKey)
	if spreadThresholdPercent == "" {
		return ArbitrageConfig{}, fmt.Errorf("%s not set", spreadPercentThresholdEnvKey)
	}

	spreadThreshold, err := sdk.NewDecFromStr(spreadThresholdPercent)
	if err != nil {
		return ArbitrageConfig{}, err
	}

	return ArbitrageConfig{
		Base:                   baseConfig,
		SpreadPercentThreshold: spreadThreshold,
	}, nil
}

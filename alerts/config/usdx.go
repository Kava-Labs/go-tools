package config

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
)

type UsdxConfig struct {
	BaseConfig
	// US dollar absolute value of how far it can stray from USDT
	UsdxDeviation sdk.Dec
	// Base USDX value which deviation should spread from
	UsdxBasePrice sdk.Dec
}

// LoadUsdxConfig loads key values from a ConfigLoader and returns a new UsdxConfig
func LoadUsdxConfig(loader ConfigLoader, logger log.Logger) (UsdxConfig, error) {
	baseConfig, err := LoadBaseConfig(loader, logger)
	if err != nil {
		return UsdxConfig{}, err
	}

	usdxDeviation := loader.Get(usdxDeviationEnvKey)
	usdxDeviationDec, err := sdk.NewDecFromStr(usdxDeviation)
	if err != nil {
		return UsdxConfig{}, err
	}

	usdxBasePrice := loader.Get(usdxBasePriceKey)
	usdxBasePriceDec, err := sdk.NewDecFromStr(usdxBasePrice)
	if err != nil {
		return UsdxConfig{}, err
	}

	return UsdxConfig{
		BaseConfig:    baseConfig,
		UsdxDeviation: usdxDeviationDec,
		UsdxBasePrice: usdxBasePriceDec,
	}, nil
}

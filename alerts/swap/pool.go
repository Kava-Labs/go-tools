package swap

import (
	"fmt"
	"math/big"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/tendermint/tendermint/libs/log"
)

// AssetSpread defines the percent spread between a Kava Swap pool and an
// external (Binance) value, the external (Binance) USD value, and the
// calculated pool USD value.
type AssetSpread struct {
	Name             string
	SpreadPercent    sdk.Dec
	ExternalUsdValue sdk.Dec
	PoolUsdValue     sdk.Dec
}

// String returns a formatted string of AssetSpread
func (a AssetSpread) String() string {
	fmtDec := func(d sdk.Dec) string {
		s := d.String()

		// 18 decimal places in Dec
		return s[:len(s)-16]
	}

	return fmt.Sprintf(
		"\t`%v`: `$%v` (pool) vs `$%v` (Binance) (`%v%%`)",
		a.Name,
		fmtDec(a.PoolUsdValue),
		fmtDec(a.ExternalUsdValue),
		fmtDec(a.SpreadPercent.MulInt64(100)),
	)
}

// PoolSpread defines the spread of a Kava Swap pool with the spread of both assets
type PoolSpread struct {
	PoolName       string
	ASpreadPercent AssetSpread
	BSpreadPercent AssetSpread
}

// String returns a formatted string of a pool and both of it's spreads
func (ps PoolSpread) String() string {
	return fmt.Sprintf(
		"Pool `%v` spread:\n\t%v\n\t%v",
		ps.PoolName,
		ps.ASpreadPercent,
		ps.BSpreadPercent,
	)
}

// ExceededThreshold returns true if a pool has exceeded a given spread threshold
func (ps PoolSpread) ExceededThreshold(threshold sdk.Dec) bool {
	return ps.ASpreadPercent.SpreadPercent.Abs().GTE(threshold) ||
		ps.BSpreadPercent.SpreadPercent.Abs().GTE(threshold)
}

// PoolSpreads defines an array of PoolSpread
type PoolSpreads []PoolSpread

// PercentChange returns the percent change from a to b.
// This will return a percent, 1 being 100%.
// This will also return a negative value if a is less than b
func PercentChange(a sdk.Dec, b sdk.Dec) (sdk.Dec, error) {
	if a.IsZero() {
		return sdk.Dec{}, fmt.Errorf("First input cannot be zero")
	}

	// (b - a) / |a|
	return b.Sub(a).Quo(a.Abs()), nil
}

// GetCoinConversionDivisor returns an Int to divide by the chain value to get the quantity
func GetCoinConversionDivisor(pools SwapPoolsData, coin sdk.Coin) sdk.Int {
	marketEntry, ok := pools.ConversionFactors[coin.Denom]
	if ok {
		i := big.NewInt(10)
		return sdk.NewIntFromBigInt(i.Exp(i, marketEntry.BigInt(), nil))
	}

	return sdk.NewIntFromBigInt(big.NewInt(1))
}

// GetPoolAssetUsdPrice returns the USD value of the first coin parameter
func GetPoolAssetUsdPrice(pools SwapPoolsData, a sdk.Coin, b sdk.Coin, bUsdValue sdk.Dec) (sdk.Dec, error) {
	if b.Amount.IsZero() {
		return sdk.Dec{}, fmt.Errorf("Cannot get price with second value 0")
	}

	aTrueAmount := a.Amount.ToDec().Quo(GetCoinConversionDivisor(pools, a).ToDec())
	bTrueAmount := b.Amount.ToDec().Quo(GetCoinConversionDivisor(pools, b).ToDec())

	// B / A == Output of B equivalent to 1 A
	// Output of B * USD price of B == USD price of 1 A
	return bTrueAmount.Quo(aTrueAmount).Mul(bUsdValue), nil
}

// GetPoolSpreads returns an array of spreads for all of the provided pools
func GetPoolSpreads(logger log.Logger, pools SwapPoolsData) (PoolSpreads, error) {
	var spreads PoolSpreads

	for _, pool := range pools.Pools {
		if len(pool.Coins) < 2 {
			logger.Error(fmt.Sprintf("Pool %v does not contain 2 coins", pool.Name))
			continue
		}

		first := pool.Coins[0]
		second := pool.Coins[1]

		// Get USD prices of coins from external sources via pricefeed or Binance
		firstUsdExternalPrice, err := GetUsdPrice(first.Denom, pools)
		if err != nil {
			// Skip the pools that have assets we cannot find USD price for
			logger.Error(err.Error())
			continue
		}

		secondUsdExternalPrice, err := GetUsdPrice(second.Denom, pools)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		// Calculate USD value of pool assets
		// Skip pools if a value is zero
		firstPoolPrice, err := GetPoolAssetUsdPrice(pools, first, second, secondUsdExternalPrice)
		if err != nil {
			logger.Error(err.Error())
			continue
		}
		secondPoolPrice, err := GetPoolAssetUsdPrice(pools, second, first, firstUsdExternalPrice)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		// Find change between external price and pool price
		// Skip this pool if there is a zero for one of the assets
		firstChange, err := PercentChange(firstPoolPrice, firstUsdExternalPrice)
		if err != nil {
			continue
		}

		secondChange, err := PercentChange(secondPoolPrice, secondUsdExternalPrice)
		if err != nil {
			continue
		}

		spread := PoolSpread{
			PoolName: pool.Name,
			ASpreadPercent: AssetSpread{
				Name:             first.Denom,
				SpreadPercent:    firstChange,
				ExternalUsdValue: firstUsdExternalPrice,
				PoolUsdValue:     firstPoolPrice,
			},
			BSpreadPercent: AssetSpread{
				Name:             second.Denom,
				SpreadPercent:    secondChange,
				ExternalUsdValue: secondUsdExternalPrice,
				PoolUsdValue:     secondPoolPrice,
			},
		}

		spreads = append(spreads, spread)
	}

	return spreads, nil
}

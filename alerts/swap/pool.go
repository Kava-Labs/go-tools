package swap

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AssetSpread struct {
	Name             string
	SpreadPercent    sdk.Dec
	ExternalUsdValue sdk.Dec
	PoolUsdValue     sdk.Dec
}

type PoolSpread struct {
	PoolName       string
	ASpreadPercent AssetSpread
	BSpreadPercent AssetSpread
}

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

// GetPoolAssetUsdPrice returns the USD value of the first coin parameter
func GetPoolAssetUsdPrice(a sdk.Coin, b sdk.Coin, bUsdValue sdk.Dec) (sdk.Dec, error) {
	if b.Amount.IsZero() {
		return sdk.Dec{}, fmt.Errorf("Cannot get price with second value 0")
	}

	// B / A == Output of B equivalent to 1 A
	// Output of B * USD price of B == USD price of 1 A
	return b.Amount.ToDec().Quo(a.Amount.ToDec()).Mul(bUsdValue), nil
}

func GetPoolSpreads(pools SwapPoolsData) (PoolSpreads, error) {
	spreads := make(PoolSpreads, 0)

	for _, pool := range pools.Pools {
		if len(pool.Coins) < 2 {
			return nil, fmt.Errorf("Pool %v does not contain 2 coins", pool.Name)
		}

		first := pool.Coins[0]
		second := pool.Coins[1]

		// Get USD prices of coins from external sources via pricefeed or Binance
		firstUsdExternalPrice, err := GetUsdPrice(first.Denom, pools)
		if err != nil {
			return nil, err
		}

		secondUsdExternalPrice, err := GetUsdPrice(second.Denom, pools)
		if err != nil {
			return nil, err
		}

		// Calculate USD value of pool assets
		firstPoolPrice, err := GetPoolAssetUsdPrice(first, second, secondUsdExternalPrice)
		if err != nil {
			return nil, err
		}
		secondPoolPrice, err := GetPoolAssetUsdPrice(second, first, firstUsdExternalPrice)
		if err != nil {
			return nil, err
		}

		fmt.Println(fmt.Sprintf("%v: secondPoolPrice %v / %v * $%v == %v",
			second.Denom,
			second.Amount,
			first.Amount,
			firstUsdExternalPrice,
			secondPoolPrice,
		))

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

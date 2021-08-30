package swap

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

type percentChangeValue struct {
	a   int64
	b   int64
	exp int64
}

func TestPercentChange(t *testing.T) {
	// Expected values checked against an online calculator
	values := []percentChangeValue{
		{5, 5, 0},
		{5, 1, -80},      // 5 -> 1
		{1, 5, 400},      // 1 -> 5
		{500, 5000, 900}, // 500 -> 5000
		{500, 0, -100},   // 500 -> 0
		{0, 1, 1},        // Should fail if first is zero
	}

	for _, val := range values {
		diff, err := PercentChange(sdk.NewDec(val.a), sdk.NewDec(val.b))
		if err != nil {
			// Should only error if the first value is 0
			assert.Equal(t, int64(0), val.a)
			continue
		}

		// Multiply by 100 such that 100 == 100%
		assert.Equal(t, val.exp, diff.MulInt64(100).RoundInt64())
	}
}

type poolAssetValue struct {
	a    sdk.Coin
	b    sdk.Coin
	bUsd sdk.Dec
	exp  int64
}

func TestGetPoolAssetPrice(t *testing.T) {
	values := []poolAssetValue{
		{
			sdk.Coin{Denom: "usd", Amount: sdk.NewInt(10)}, sdk.Coin{Denom: "usd", Amount: sdk.NewInt(10)},
			sdk.NewDec(1), 1,
		},
		{
			sdk.Coin{Denom: "bnb", Amount: sdk.NewInt(100000000)}, sdk.Coin{Denom: "usdx", Amount: sdk.NewInt(492000000)},
			sdk.NewDec(1), 492,
		},
		// Some actual pool values
		{
			sdk.Coin{Denom: "bnb", Amount: sdk.NewInt(19247262658)}, sdk.Coin{Denom: "usdx", Amount: sdk.NewInt(91486075424)},
			sdk.NewDec(1), 475,
		},
		{
			sdk.Coin{Denom: "btcb", Amount: sdk.NewInt(495726043)}, sdk.Coin{Denom: "usdx", Amount: sdk.NewInt(239260237273)},
			sdk.NewDec(1), 48265,
		},
	}

	poolData := SwapPoolsData{
		CdpMarkets: map[string]sdk.Int{
			"bnb":  sdk.NewInt(8),
			"usdx": sdk.NewInt(6),
			"btcb": sdk.NewInt(8),
		},
	}

	for _, val := range values {
		usdValue, err := GetPoolAssetUsdPrice(poolData, val.a, val.b, val.bUsd)
		if err != nil {
			// Should only error if the first value is 0
			assert.Equal(t, int64(0), val.a)
			continue
		}

		assert.Equal(t, val.exp, usdValue.RoundInt64())
	}
}

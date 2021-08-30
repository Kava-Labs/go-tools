package swap

import (
	"fmt"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/assert"
)

func TestPercentChange(t *testing.T) {
	// Expected values checked against an online calculator
	values := []struct {
		giveAValue        int64
		giveBValue        int64
		wantPercentChange int64
	}{
		{5, 5, 0},
		{5, 1, -80},      // 5 -> 1
		{1, 5, 400},      // 1 -> 5
		{500, 5000, 900}, // 500 -> 5000
		{500, 0, -100},   // 500 -> 0
		{0, 1, 1},        // Should fail if first is zero
	}

	for _, tt := range values {
		t.Run(fmt.Sprintf("%v->%v", tt.giveAValue, tt.giveBValue), func(t *testing.T) {
			diff, err := PercentChange(sdk.NewDec(tt.giveAValue), sdk.NewDec(tt.giveBValue))
			if err != nil {
				// Should only error if the first value is 0
				assert.Equal(t, int64(0), tt.giveAValue)
				return
			}

			// Multiply by 100 such that 100 == 100%
			assert.Equal(t, tt.wantPercentChange, diff.MulInt64(100).RoundInt64())
		})
	}
}

func TestGetPoolAssetPrice(t *testing.T) {
	values := []struct {
		giveACoin  sdk.Coin
		giveBCoin  sdk.Coin
		giveBUsd   sdk.Dec
		wantAPrice int64
	}{
		{
			sdk.Coin{Denom: "usd", Amount: sdk.NewInt(10)},
			sdk.Coin{Denom: "usd", Amount: sdk.NewInt(10)},
			sdk.NewDec(1),
			1,
		},
		{
			sdk.Coin{Denom: "bnb", Amount: sdk.NewInt(100000000)},
			sdk.Coin{Denom: "usdx", Amount: sdk.NewInt(492000000)},
			sdk.NewDec(1),
			492,
		},
		// Some actual pool values
		{
			sdk.Coin{Denom: "bnb", Amount: sdk.NewInt(19247262658)},
			sdk.Coin{Denom: "usdx", Amount: sdk.NewInt(91486075424)},
			sdk.NewDec(1),
			475,
		},
		{
			sdk.Coin{Denom: "btcb", Amount: sdk.NewInt(495726043)},
			sdk.Coin{Denom: "usdx", Amount: sdk.NewInt(239260237273)},
			sdk.NewDec(1),
			48265,
		},
	}

	poolData := SwapPoolsData{
		ConversionFactors: map[string]sdk.Int{
			"bnb":  sdk.NewInt(8),
			"usdx": sdk.NewInt(6),
			"btcb": sdk.NewInt(8),
		},
	}

	for _, tt := range values {
		t.Run(fmt.Sprintf("%v:%v", tt.giveACoin, tt.giveBCoin), func(t *testing.T) {
			usdValue, err := GetPoolAssetUsdPrice(poolData, tt.giveACoin, tt.giveBCoin, tt.giveBUsd)
			if err != nil {
				// Should only error if the first value is 0
				assert.Equal(t, int64(0), tt.giveACoin)
				return
			}

			assert.Equal(t, tt.wantAPrice, usdValue.RoundInt64())
		})
	}
}

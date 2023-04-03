package main

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/auction-audit/types"
)

// GetTotalCoinsUsdValueAtHeight returns the total price of a slice of coins in
// USD at the given height.
func GetTotalCoinsUsdValueAtHeight(
	client Client,
	height int64,
	coins sdk.Coins,
	priceType types.PriceType,
) (sdk.Dec, error) {
	totalUsdValue := sdk.ZeroDec()

	for _, coin := range coins {
		marketId, err := types.GetMarketID(coin.Denom, priceType)
		if err != nil {
			return sdk.ZeroDec(), fmt.Errorf("could not find market id for denom %s: %w", coin.Denom, err)
		}

		price, err := client.GetPriceAtHeight(height, marketId)
		if err != nil {
			return sdk.ZeroDec(), fmt.Errorf("failed to get price for %s at height %d", marketId, height)
		}

		conversionFactor, found := types.ConversionMap[coin.Denom]
		if !found {
			return sdk.ZeroDec(), fmt.Errorf("could not find conversion factor for denom %s", coin.Denom)
		}

		coinValue := coin.Amount.ToDec().Quo(conversionFactor.ToDec()).Mul(price)
		totalUsdValue = totalUsdValue.Add(coinValue)
	}

	return totalUsdValue, nil
}

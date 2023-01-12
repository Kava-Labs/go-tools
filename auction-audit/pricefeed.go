package main

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/auction-audit/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
)

// GetPriceAtHeight returns the price of the given market at the given height.
func GetPriceAtHeight(client GrpcClient, height int64, marketID string) (sdk.Dec, error) {
	ctx := ctxAtHeight(height)

	res, err := client.Pricefeed.Price(
		ctx,
		&pricefeedtypes.QueryPriceRequest{
			MarketId: marketID,
		},
	)

	if err != nil {
		return sdk.ZeroDec(), err
	}

	return res.Price.Price, nil
}

// GetTotalCoinsUsdValueAtHeight returns the total price of a slice of coins in
// USD at the given height.
func GetTotalCoinsUsdValueAtHeight(
	client GrpcClient,
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

		price, err := GetPriceAtHeight(client, height, marketId)
		if err != nil {
			return sdk.ZeroDec(), fmt.Errorf("failed to get price for %s at height %d", marketId, height)
		}

		coinValue := coin.Amount.ToDec().Mul(price)
		totalUsdValue = totalUsdValue.Add(coinValue)
	}

	return totalUsdValue, nil
}

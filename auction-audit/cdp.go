package main

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type AttributesMap map[string]string

func GetOriginalAmountPercentSub(
	percentSubtracted sdk.Dec,
	afterAmount sdk.Dec,
) sdk.Dec {
	// Convert subtracted percent to total percent (5% fee becomes 95% of orig)
	return afterAmount.
		// Convert percent to whole number, find 1% of original value
		Quo(sdk.NewDec(100).Sub(percentSubtracted.Mul(sdk.NewDec(100)))).
		// Multiply by 100 to get 100% of original value
		Mul(sdk.NewDec(100))
}

func GetAuctionStartLotCDP(
	ctx context.Context,
	client Client,
	auctionID uint64,
) (sdk.Coin, int64, error) {
	// Search BeginBlock events for CDP
	data, found := auctionStartIndex[auctionID]
	if found {
		height, err := strconv.ParseInt(data["height"], 10, 64)
		if err != nil {
			return sdk.Coin{}, 0, err
		}

		lot, err := sdk.ParseCoin(data["lot"])
		if err != nil {
			return sdk.Coin{}, 0, err
		}

		return lot, height, nil
	}

	// Try searching for liquidate message in Txs
	lot, height, err := GetAuctionStartLotTxResponses(ctx, client, auctionID)
	if err != nil {
		return sdk.Coin{}, 0, fmt.Errorf("failed to get CDP auction start from both BeginBlock and TxSearch: %w", err)
	}

	return lot, height, nil
}

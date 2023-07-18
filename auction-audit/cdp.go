package main

import (
	"context"
	"fmt"

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
	client GrpcClient,
	auctionID uint64,
) (sdk.Coin, int64, error) {
	// Search BeginBlock events for CDP
	blockEvents, height, err := client.GetBeginBlockEventsFromQuery(
		ctx,
		fmt.Sprintf(
			"auction_start.auction_id=%d",
			auctionID,
		))

	// No error: auction started in BeginBlocker
	if err == nil {
		// Check if auction started in BeginBlocker, continue if not found
		lot, found := GetAuctionStartLotFromEvents(blockEvents, auctionID)
		if !found {
			return sdk.Coin{}, 0, fmt.Errorf("failed to get CDP auction start from BeginBlock: %s", err)
		}

		// This should exist at this point, beginblock query will return an error
		// if the event for the specified auction was not found
		return lot, height, nil
	}

	// Try searching for liquidate message in Txs
	lot, height, err := GetAuctionStartLotTxResponses(ctx, client, auctionID)
	if err != nil {
		return sdk.Coin{}, 0, fmt.Errorf("failed to get CDP auction start from both BeginBlock and TxSearch: %w", err)
	}

	return lot, height, nil
}

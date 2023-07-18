package main

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

func GetAuctionSourceHARD(
	ctx context.Context,
	client GrpcClient,
	auctionID uint64,
) (sdk.Coin, int64, error) {
	lot, height, err := GetAuctionStartLotTxResponses(ctx, client, auctionID)
	if err != nil {
		return sdk.Coin{}, 0, fmt.Errorf("failed to get hard auction start from both BeginBlock and TxSearch: %w", err)
	}

	return lot, height, nil
}

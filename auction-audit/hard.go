package main

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
)

func GetAuctionSourceHARD(
	ctx context.Context,
	client GrpcClient,
	auctionID int64,
) (hardtypes.DepositResponse, error) {
	res, err := client.Tx.GetTxsEvent(
		ctx,
		&tx.GetTxsEventRequest{
			Events: []string{
				"message.action='/kava.hard.v1beta1.MsgLiquidate'",
				fmt.Sprintf(
					"auction_start.auction_id=%d",
					auctionID,
				),
			},
		},
	)
	if err != nil {
		return hardtypes.DepositResponse{}, err
	}

	var pairs []CDPAuctionPair

	// Get corresponding CDP from liquidate event
	for _, tsRes := range res.TxResponses {
		// There can be multiple liquidations in a single block
		// TODO: There can be multiple liquidations per TxResponse (multiple msgs in 1 tx)
		// TODO: How to match them confirming same amount?
		pair, err := getCdpAuctionPairFromTxResponse(tsRes)
		if err != nil {
			// There must be a matching event in every TxResponse, as we are
			// querying for the matching event.
			return cdptypes.CDPResponse{}, err
		}

		pairs = append(pairs, pair)
	}

	// Find matching CDP
	var matchingPair CDPAuctionPair
	found := false
	for _, pair := range pairs {
		if pair.AuctionId.Equal(sdk.NewInt(auctionID)) {
			matchingPair = pair
			found = true
		}
	}

	if !found {
		return hardtypes.DepositResponse{}, fmt.Errorf("failed to find matching hard deposit for auction ID %d", auctionID)
	}

	// Query CDP at height 1 before liquidation, as it is deleted when liquidated
	return GetCDPAtHeight(client, matchingPair.Height-1, matchingPair.CdpId.Uint64())
}

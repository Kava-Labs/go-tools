package main

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
)

type HardAuctionPair struct {
	Height    int64
	Address   sdk.AccAddress
	AuctionId sdk.Int
}

type HardAuctionPairs []HardAuctionPair

func (pairs HardAuctionPairs) FindPairWithAuctionID(auctionID sdk.Int) (HardAuctionPair, bool) {
	for _, pair := range pairs {
		if pair.AuctionId.Equal(auctionID) {
			return pair, true
		}
	}

	return HardAuctionPair{}, false
}

func getAddressFromEvent(attrs []sdk.Attribute) (sdk.AccAddress, error) {
	attrsMap := AttributesToMap(attrs)

	depositorStr, found := attrsMap[hardtypes.AttributeKeyLiquidatedOwner]
	if !found {
		return sdk.AccAddress{}, fmt.Errorf("failed to find liquidated_owner in hard_liquidation event")
	}

	addr, err := sdk.AccAddressFromBech32(depositorStr)
	if err != nil {
		return sdk.AccAddress{}, fmt.Errorf("failed to parse liquidated_owner address: %s", depositorStr)
	}

	return addr, nil
}

func getHardAuctionPairs(txResponses []*sdk.TxResponse) (HardAuctionPairs, error) {
	var pairs HardAuctionPairs

	// Get corresponding CDP from liquidate event
	for _, tsRes := range txResponses {
		// There can be multiple liquidations in a single block
		// TODO: There can be multiple liquidations per TxResponse (multiple msgs in 1 tx)
		// TODO: How to match them confirming same amount?
		pair, err := getHardAuctionPairFromTxResponse(tsRes)
		if err != nil {
			// There must be a matching event in every TxResponse, as we are
			// querying for the matching event.
			return nil, err
		}

		pairs = append(pairs, pair)
	}

	return pairs, nil
}

func getHardAuctionPairFromTxResponse(txResponse *sdk.TxResponse) (HardAuctionPair, error) {
	var ownerAddr sdk.AccAddress
	var auctionId sdk.Int
	ownerAddrFound := false
	auctionIdFound := false

	for _, log := range txResponse.Logs {
		for _, event := range log.Events {
			// Only interested in cdp liquidation and auction start events
			// TODO: Unhandled edge case: there can be multiple of these events
			// per TxResponse if there are multiple Liquidate messages in the tx.
			// OR there could be 1 event each but with multiple copies of the
			// attributes -- only the last one will be returned
			if event.Type == hardtypes.EventTypeHardLiquidation {
				addr, err := getAddressFromEvent(event.Attributes)
				if err != nil {
					return HardAuctionPair{}, err
				}

				ownerAddr = addr
				ownerAddrFound = true
			}

			if event.Type == auctiontypes.EventTypeAuctionStart {
				id, err := getAuctionIDFromAuctionStartEvent(event.Attributes)
				if err != nil {
					return HardAuctionPair{}, err
				}

				auctionId = id
				auctionIdFound = true
			}
		}
	}

	if !ownerAddrFound || !auctionIdFound {
		return HardAuctionPair{},
			fmt.Errorf("failed to find hard and/or auction id in tx response: %s", txResponse.TxHash)
	}

	return HardAuctionPair{
		Address:   ownerAddr,
		AuctionId: auctionId,
		Height:    txResponse.Height,
	}, nil
}

func GetHardDepositAtHeight(
	client GrpcClient,
	height int64,
	addr sdk.AccAddress,
) (hardtypes.DepositResponse, error) {
	queryCtx := ctxAtHeight(height)

	res, err := client.Hard.Deposits(queryCtx, &hardtypes.QueryDepositsRequest{
		Owner: addr.String(),
	})

	if err != nil {
		return hardtypes.DepositResponse{}, err
	}

	if len(res.Deposits) == 0 {
		return hardtypes.DepositResponse{}, fmt.Errorf("hard deposit for %s was not found at block %d", addr, height)
	}

	return res.Deposits[0], nil
}

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

	pairs, err := getHardAuctionPairs(res.TxResponses)
	if err != nil {
		return hardtypes.DepositResponse{}, err
	}

	// Find matching address
	matchingPair, found := pairs.FindPairWithAuctionID(sdk.NewInt(auctionID))
	if !found {
		return hardtypes.DepositResponse{}, fmt.Errorf("failed to find matching hard deposit for auction ID %d", auctionID)
	}

	// Deposit deleted when liquidated, query at previous block
	return GetHardDepositAtHeight(client, matchingPair.Height-1, matchingPair.Address)
}

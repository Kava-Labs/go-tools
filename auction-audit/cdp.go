package main

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
)

type CDPAuctionPair struct {
	Height    int64
	CdpId     sdk.Int
	AuctionId sdk.Int
}

type CDPAuctionPairs []CDPAuctionPair

func (pairs CDPAuctionPairs) FindPairWithAuctionID(auctionID sdk.Int) (CDPAuctionPair, bool) {
	for _, pair := range pairs {
		if pair.AuctionId.Equal(auctionID) {
			return pair, true
		}
	}

	return CDPAuctionPair{}, false
}

type AttributesMap map[string]string

func AttributesToMap(attrs []sdk.Attribute) AttributesMap {
	m := make(map[string]string)
	for _, attr := range attrs {
		m[attr.Key] = attr.Value
	}

	return m
}

func getCdpIDFromLiquidationEvent(attrs []sdk.Attribute) (sdk.Int, error) {
	attrsMap := AttributesToMap(attrs)

	idStr, found := attrsMap[cdptypes.AttributeKeyCdpID]
	if !found {
		return sdk.Int{}, fmt.Errorf("failed to find cdp id in liquidation event")
	}

	id, ok := sdk.NewIntFromString(idStr)
	if !ok {
		return sdk.Int{}, fmt.Errorf("failed to parse cdp id: %s", idStr)
	}

	return id, nil
}

func getAuctionIDFromAuctionStartEvent(attrs []sdk.Attribute) (sdk.Int, error) {
	attrsMap := AttributesToMap(attrs)

	idStr, found := attrsMap[auctiontypes.AttributeKeyAuctionID]
	if !found {
		return sdk.Int{}, fmt.Errorf("failed to find auction id in auction start event")
	}

	id, ok := sdk.NewIntFromString(idStr)
	if !ok {
		return sdk.Int{}, fmt.Errorf("failed to parse auction id: %s", idStr)
	}

	return id, nil
}

func getCdpAuctionPairFromTxResponse(txResponse *sdk.TxResponse) (CDPAuctionPair, error) {
	var cdpId sdk.Int
	var auctionId sdk.Int
	cdpIdFound := false
	auctionIdFound := false

	for _, log := range txResponse.Logs {
		for _, event := range log.Events {
			// Only interested in cdp liquidation and auction start events
			// TODO: Unhandled edge case: there can be multiple of these events
			// per TxResponse if there are multiple Liquidate messages in the tx.
			// OR there could be 1 event each but with multiple copies of the
			// attributes -- only the last one will be returned
			if event.Type == cdptypes.EventTypeCdpLiquidation {
				id, err := getCdpIDFromLiquidationEvent(event.Attributes)
				if err != nil {
					return CDPAuctionPair{}, err
				}

				cdpId = id
				cdpIdFound = true
			}

			if event.Type == auctiontypes.EventTypeAuctionStart {
				id, err := getAuctionIDFromAuctionStartEvent(event.Attributes)
				if err != nil {
					return CDPAuctionPair{}, err
				}

				auctionId = id
				auctionIdFound = true
			}
		}
	}

	if !cdpIdFound || !auctionIdFound {
		return CDPAuctionPair{},
			fmt.Errorf("failed to find cdp and/or auction id in tx response: %s", txResponse.TxHash)
	}

	return CDPAuctionPair{
		CdpId:     cdpId,
		AuctionId: auctionId,
		Height:    txResponse.Height,
	}, nil
}

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

func GetCDPAtHeight(
	client GrpcClient,
	height int64,
	cdpId uint64,
) (cdptypes.CDPResponse, error) {
	queryCtx := ctxAtHeight(height)

	// Fetch CDP to determine original value prior to liquidation
	cdpRes, err := client.CDP.Cdps(queryCtx, &cdptypes.QueryCdpsRequest{
		ID: cdpId,
	})

	if err != nil {
		return cdptypes.CDPResponse{}, err
	}

	if len(cdpRes.Cdps) == 0 {
		return cdptypes.CDPResponse{}, fmt.Errorf("cdp %d was not found at block %d", cdpId, height)
	}

	return cdpRes.Cdps[0], nil
}

func GetAuctionSourceCDP(
	ctx context.Context,
	client GrpcClient,
	auctionID uint64,
) (cdptypes.CDPResponse, int64, error) {
	// TODO: Search BeginBlock events for CDP
	// 1) Block search to find auction_start event and corresponding height
	// https://rpc.kava.io/block_search?query=%22auction_start.auction_id=16837%22
	// 2) Block results to query events from height
	// https://rpc.kava.io/block_results?height=3146803

	// This is not very common for manual liquidations, most liquidations are in
	// CDP BeginBlocker
	res, err := client.Tx.GetTxsEvent(
		ctx,
		&tx.GetTxsEventRequest{
			Events: []string{
				// Query service joins multiple events in request slice with AND
				"cdp_liquidation.module='cdp'",
				fmt.Sprintf(
					"auction_start.auction_id=%d",
					auctionID,
				),
			},
		},
	)
	if err != nil {
		return cdptypes.CDPResponse{}, 0, err
	}

	var pairs CDPAuctionPairs

	// Get corresponding CDP from liquidate event
	for _, tsRes := range res.TxResponses {
		// There can be multiple liquidations in a single block
		// TODO: There can be multiple liquidations per TxResponse (multiple msgs in 1 tx)
		// TODO: How to match them confirming same amount?
		pair, err := getCdpAuctionPairFromTxResponse(tsRes)
		if err != nil {
			// There must be a matching event in every TxResponse, as we are
			// querying for the matching event.
			return cdptypes.CDPResponse{}, 0, err
		}

		pairs = append(pairs, pair)
	}

	// Get the corresponding CDP ID with the auction ID
	matchingPair, found := pairs.FindPairWithAuctionID(sdk.NewIntFromUint64(auctionID))

	if !found {
		return cdptypes.CDPResponse{}, 0, fmt.Errorf("failed to find matching CDP for auction ID %d", auctionID)
	}

	// Query CDP at height 1 before liquidation, as it is deleted when liquidated
	depositHeight := matchingPair.Height - 1

	cdp, err := GetCDPAtHeight(client, depositHeight, matchingPair.CdpId.Uint64())
	return cdp, depositHeight, err
}

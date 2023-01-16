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

type CDPAuctionPairWitHeight struct {
	CDPAuctionPair
	Height int64
}

type CDPAuctionPairsWitHeight []CDPAuctionPairsWitHeight

type AttributesMap map[string]string

func AttributesToMap(attrs []sdk.Attribute) AttributesMap {
	m := make(map[string]string)
	for _, attr := range attrs {
		if _, found := m[attr.Key]; found {
			panic(fmt.Sprintf("duplicate attribute key %s, this may be due to flattened events", attr.Key))
		}

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

func getCdpAuctionPairFromEvents(
	height int64,
	events sdk.StringEvents,
) (CDPAuctionPairs, error) {
	cdpIdFound := false
	auctionIdFound := false

	var pairs CDPAuctionPairs
	pair := CDPAuctionPair{
		Height: height,
	}

	// Assumptions:
	// 1) Events are **not** flattened
	// 2) Events are in the order of execution, multiple liquidations
	//    will have cdp_liquidation and auction_start in order of liquidation.
	//
	//    Example:
	//    - cdp_liquidation             (cdp 1)
	//    - misc transfers and whatnot
	//    - auction_start               (cdp 1)
	//    - other misc events
	//    - cdp_liquidation             (cdp 2)
	//    - misc transfers and whatnot
	//    - auction_start               (cdp 2)
	for _, event := range events {
		if event.Type == cdptypes.EventTypeCdpLiquidation {
			// Invalid situation: Found a second cdp_liquidation without an
			// auction_start event in between
			if cdpIdFound {
				return nil, fmt.Errorf("found cdp_liquidation event without corresponding auction_start event")
			}

			id, err := getCdpIDFromLiquidationEvent(event.Attributes)
			if err != nil {
				return nil, err
			}

			pair.CdpId = id
			cdpIdFound = true
		}

		if event.Type == auctiontypes.EventTypeAuctionStart {
			// Invalid situation: Found a second auction_start without a
			// cdp_liquidation event in between
			if auctionIdFound {
				return nil, fmt.Errorf("found auction_start event without corresponding cdp_liquidation event")
			}

			id, err := getAuctionIDFromAuctionStartEvent(event.Attributes)
			if err != nil {
				return nil, err
			}

			pair.AuctionId = id
			auctionIdFound = true
		}

		// Found both cdp and auction id, create a pair and reset flags for next pair
		if cdpIdFound && auctionIdFound {
			pairs = append(pairs, pair)

			pair = CDPAuctionPair{
				Height: height,
			}
			cdpIdFound = false
			auctionIdFound = false
		}
	}

	if len(pairs) == 0 {
		return nil, fmt.Errorf("could not find cdp and/or auction id in events")
	}

	return pairs, nil
}

func getCdpAuctionPairsFromTxResponse(txResponse *sdk.TxResponse) (CDPAuctionPairs, error) {
	var pairs CDPAuctionPairs

	for _, log := range txResponse.Logs {
		// Separate log per message, so this should handle multiple liquidate
		// messages
		foundPairs, err := getCdpAuctionPairFromEvents(txResponse.Height, log.Events)
		if err != nil {
			continue
		}

		pairs = append(pairs, foundPairs...)
	}

	// Add height to pairs
	for i := range pairs {
		pairs[i].Height = txResponse.Height
	}

	return pairs, nil
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
		return cdptypes.CDPResponse{}, fmt.Errorf("failed to query Cdps: %w", err)
	}

	if len(cdpRes.Cdps) == 0 {
		return cdptypes.CDPResponse{}, fmt.Errorf("cdp %d was not found at block %d", cdpId, height)
	}

	return cdpRes.Cdps[0], nil
}

func GetPairsFromTxSearch(
	ctx context.Context,
	client GrpcClient,
	auctionID uint64,
) (CDPAuctionPairs, error) {
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
		return nil, err
	}

	var pairs CDPAuctionPairs

	// Get corresponding CDP from liquidate event
	for _, tsRes := range res.TxResponses {
		// There can be multiple liquidations in a single block
		foundPairs, err := getCdpAuctionPairsFromTxResponse(tsRes)
		if err != nil {
			// There must be a matching event in every TxResponse, as we are
			// querying for the matching event.
			return nil, fmt.Errorf("failed to find matching cdp_liquidation event: %w", err)
		}

		pairs = append(pairs, foundPairs...)
	}

	return pairs, nil
}

func GetAuctionSourceCDP(
	ctx context.Context,
	client GrpcClient,
	auctionID uint64,
) (cdptypes.CDPResponse, int64, error) {
	// Search BeginBlock events for CDP

	blockEvents, height, err := client.GetBeginBlockEventsFromQuery(
		ctx,
		fmt.Sprintf(
			"auction_start.auction_id=%d",
			auctionID,
		))

	var pairs CDPAuctionPairs
	if err != nil {
		// Try searching Txs for event instead
		pairs, err = GetPairsFromTxSearch(ctx, client, auctionID)
		if err != nil {
			return cdptypes.CDPResponse{}, 0, fmt.Errorf("failed to get CDP auction pairs from both BeginBlock and TxSearch: %w", err)
		}
	} else {
		// BeginBlock events are all in 1 single large slice, so there can be
		// multiple of the same events in the slice
		pairs, err = getCdpAuctionPairFromEvents(height, blockEvents)
		if err != nil {
			return cdptypes.CDPResponse{}, 0, fmt.Errorf("failed to get CDP auction pairs from block events: %w", err)
		}
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

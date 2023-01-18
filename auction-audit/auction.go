package main

import (
	"context"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
)

type AuctionStartEventData struct {
	ID  uint64
	Lot sdk.Coin
}

type AuctionStartEventsData []AuctionStartEventData

func GetAuctionStartLotFromEvents(
	events sdk.StringEvents,
	auctionID uint64,
) (sdk.Coin, bool) {

	for _, event := range events {
		if event.Type != auctiontypes.EventTypeAuctionStart {
			continue
		}

		isMatch := false

		for _, attrs := range event.Attributes {
			if attrs.Key == auctiontypes.AttributeKeyAuctionID {
				id, err := strconv.ParseUint(attrs.Value, 10, 64)
				if err != nil {
					continue
				}

				if id != auctionID {
					continue
				}

				isMatch = true
			}

			// Only return the lot if the auction_id matches.
			// Assumption: lot attribute will always come after auction_id.
			if attrs.Key == auctiontypes.AttributeKeyLot && isMatch {
				lot, err := sdk.ParseCoinNormalized(attrs.Value)
				if err != nil {
					continue
				}

				return lot, true
			}
		}
	}

	return sdk.Coin{}, false
}

// GetAuctionStartLotTxResponses fetches an auction's start event via tx search
// event for the corresponding auction_id.
func GetAuctionStartLotTxResponses(
	ctx context.Context,
	client GrpcClient,
	auctionID uint64,
) (sdk.Coin, int64, error) {
	res, err := client.Tx.GetTxsEvent(
		ctx,
		&tx.GetTxsEventRequest{
			Events: []string{
				fmt.Sprintf(
					"auction_start.auction_id=%d",
					auctionID,
				),
			},
		},
	)
	if err != nil {
		return sdk.Coin{}, 0, fmt.Errorf("failed to query tx event: %w", err)
	}

	if len(res.TxResponses) == 0 {
		return sdk.Coin{}, 0, fmt.Errorf("no txs with auction_start found for auction ID %d", auctionID)
	}

	for _, res := range res.TxResponses {
		start, found := GetAuctionStartLotFromEvents(sdk.StringifyEvents(res.Events), auctionID)
		if found {
			return start, res.Height, nil
		}
	}

	return sdk.Coin{}, 0, fmt.Errorf("no auction start events found for auction ID %d", auctionID)
}

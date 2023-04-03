package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	"github.com/tendermint/tendermint/libs/log"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type AuctionStartEventData struct {
	ID  uint64
	Lot sdk.Coin
}

type AuctionStartEventsData []AuctionStartEventData

func GetAuctionBidEvents(
	logger log.Logger,
	client Client,
	start, end int64,
) ([]*coretypes.ResultTx, error) {
	queryEvents := []string{
		"message.action='place_bid'",
		fmt.Sprintf(
			"tx.height >= %d",
			start,
		),
		// We can technically also add tx.height <= end, but queries with it
		// almost always timeout, compared to without it.
	}

	query := strings.Join(queryEvents, " AND ")

	limit := 100

	page := 1
	var txs []*coretypes.ResultTx

pages:
	for {
		logger.Debug(
			"Querying auction bids page",
			"page", page, "limit", limit, "query", query,
		)

		// 10 attempts to query each page
		for i := 0; i < 10; i++ {
			res, err := client.Tendermint.TxSearch(
				query,
				false, // prove false, as true will require the node to fetch each block
				page,
				limit,
				"asc",
			)
			if err != nil {
				logger.Error(
					"Error querying txs, retrying",
					"err", err, "page", page, "attempt", i,
				)
				continue
			}

			logger.Debug(
				"Found auction bids",
				"page", page,
				"count", len(res.Txs),
			)

			// oldest to newest (ascending block height)
			// low    to high
			txs = append(txs, res.Txs...)

			// None in response, done
			if len(res.Txs) == 0 {
				break pages
			}

			// TODO: need to catch last page error since this won't work with exact modded pages
			if len(res.Txs) < 100 {
				break pages
			}

			// Check if end height in the **new txs** if completed
			lastTx := res.Txs[len(res.Txs)-1]

			// Last tx is greater than queried end height, done
			if lastTx.Height > end {
				logger.Debug(
					"Last page of auction bids found",
					"page", page,
					"count", len(res.Txs),
					"oldestHeight", lastTx.Height,
				)
				break pages
			}

			// Success, next page
			page += 1
			continue pages
		}

		// Failed 10 times
		return nil, fmt.Errorf("failed to query txs after 10 times")
	}

	return txs, nil
}

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
				lot, err := sdk.ParseCoin(attrs.Value)
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
	client Client,
	auctionID uint64,
) (sdk.Coin, int64, error) {
	page := 1
	limit := 100

	res, err := client.Tendermint.TxSearch(
		fmt.Sprintf(
			"auction_start.auction_id=%d",
			auctionID,
		),
		false, // prove false, as true will require the node to fetch each block
		page,
		limit,
		"asc",
	)
	if err != nil {
		return sdk.Coin{}, 0, fmt.Errorf("failed to query tx event: %w", err)
	}

	if len(res.Txs) == 0 {
		return sdk.Coin{}, 0, fmt.Errorf("no txs with auction_start found for auction ID %d", auctionID)
	}

	for _, txRes := range res.Txs {
		start, found := GetAuctionStartLotFromEvents(sdk.StringifyEvents(txRes.TxResult.Events), auctionID)
		if found {
			return start, txRes.Height, nil
		}
	}

	return sdk.Coin{}, 0, fmt.Errorf("no auction start events found for auction ID %d", auctionID)
}

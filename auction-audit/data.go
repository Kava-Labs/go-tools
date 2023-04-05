package main

import (
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/tendermint/tendermint/libs/log"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/kava-labs/go-tools/auction-audit/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	"golang.org/x/sync/errgroup"
)

const (
	concurrency        = 2
	hourSeconds        = 3600
	approxBlockSeconds = 6
)

// func GetInboundTransfers(client Client, start, end int64) (map[*sdk.AccAddress]sdk.Coins, error) {
// 	heights := make(chan int64)
// 	rawOutput := make(chan)
// }

func GetAuctionEndData(
	logger log.Logger,
	client Client,
	start, end int64,
) (types.AuctionIDToHeightMap, error) {
	// Get all auctions that ended between start and end
	auctionResultTxs, err := GetAuctionBidEvents(logger, client, start, end)

	if err != nil {
		return nil, err
	}

	// auction ID -> block height
	aucMap := make(types.AuctionIDToHeightMap)

	for _, txRes := range auctionResultTxs {
		// Query may include blocks after end height
		if txRes.Height > end {
			break
		}

		// Skip failed txs
		if txRes.TxResult.Code != 0 {
			continue
		}

		strEvents := sdk.StringifyEvents(txRes.TxResult.Events)

		// All events in a message
	msg:
		for _, event := range strEvents {
			if event.Type != auctiontypes.EventTypeAuctionBid {
				continue
			}

			// Look for auction ID
			for _, attr := range event.Attributes {
				if attr.Key == auctiontypes.AttributeKeyAuctionID {
					auctionID, err := strconv.ParseUint(string(attr.Value), 10, 64)
					if err != nil {
						return nil, err
					}

					prevHeight, found := aucMap[auctionID]
					// Saved height is older than this tx
					if found && prevHeight > txRes.Height {
						// Move to next msg, current tx might have other bids for other auctions
						continue msg
					}

					aucMap[auctionID] = txRes.Height
				}
			}
		}
	}

	return aucMap, nil
}

type auctionPair struct {
	id     uint64
	height int64
}

type auctionResponse struct {
	EndHeight int64
	Auction   auctiontypes.Auction
}

func GetAuctionClearingData(
	logger log.Logger,
	client Client,
	endMap types.AuctionIDToHeightMap,
) (types.BaseAuctionProceedsMap, error) {
	// Return if empty, otherwise the loop over rawOutput will hang forever:
	// rawOutput will never have any values to read, so the loop will never start.
	if len(endMap) == 0 {
		return types.BaseAuctionProceedsMap{}, nil
	}

	rawOutput := make(chan auctionResponse)
	pairs := make(chan auctionPair)

	clearingMap := make(types.BaseAuctionProceedsMap)

	// buffer output and order
	go func() {
	}()
	// spawn pool of workers
	for i := 0; i < concurrency; i++ {
		go fetchAuction(logger, client, pairs, rawOutput)
	}
	// write heights to input channel
	go func() {
		for id, height := range endMap {
			pairs <- auctionPair{id: id, height: height}
		}
	}()

	bar := progressbar.Default(int64(len(endMap)))

	i := 1
	for res := range rawOutput {
		if res.Auction == nil {
			logger.Error("auction response was nil", "endHeight", res.EndHeight, "id", i)
			continue
		}

		bar.Add(1)
		bar.Describe(fmt.Sprintf("Processing auction %d", res.Auction.GetID()))

		col, ok := res.Auction.(auctiontypes.CollateralAuction)
		if ok {
			ap, found := clearingMap[col.GetID()]

			if found {
				ap.AmountPurchased = ap.AmountPurchased.Add(col.GetLot())
				ap.AmountPaid = ap.AmountPaid.Add(col.GetBid())
				ap.LiquidatedAccount = col.GetLotReturns().Addresses[0].String()
				ap.WinningBidder = col.GetBidder().String()

				clearingMap[col.GetID()] = ap
			} else {
				clearingMap[col.GetID()] = types.BaseAuctionProceeds{
					ID:                col.GetID(),
					EndHeight:         res.EndHeight,
					AmountPurchased:   col.GetLot(),
					AmountPaid:        col.GetBid(),
					LiquidatedAccount: col.GetLotReturns().Addresses[0].String(),
					WinningBidder:     col.GetBidder().String(),
					SourceModule:      col.GetInitiator(),
				}
			}
		}

		if i == len(endMap) {
			break
		}
		i++
	}
	bar.Finish()

	return clearingMap, nil
}

// GetAuctionValueData populates a map with before/after auction USD value data
func GetAuctionValueData(
	ctx context.Context,
	logger log.Logger,
	client Client,
	clearingData types.BaseAuctionProceedsMap,
) (types.AuctionProceedsMap, error) {
	// Return if empty, otherwise the loop over rawOutput will hang forever:
	// rawOutput will never have any values to read, so the loop will never start.
	if len(clearingData) == 0 {
		return types.AuctionProceedsMap{}, nil
	}

	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	proceedsChan := make(chan types.AuctionProceeds)
	dataMap := make(types.AuctionProceedsMap)

	// Run worker jobs in background so that the consumer proceedsChan can
	// read from the channel as soon as the first worker job is done.
	go func() {
		bar := progressbar.Default(int64(len(clearingData)))
		for _, auctionData := range clearingData {
			func(auctionData types.BaseAuctionProceeds) {
				g.Go(func() error {
					bar.Add(1)
					bar.Describe(fmt.Sprintf("Adding USD value data to auction %d", auctionData.ID))

					// Liquidation initial lot and height of liquidation
					initialLot, preLiquidationHeight := fetchLiquidationData(ctx, logger, client, auctionData)

					// Get total USD value before liquidation
					beforeUsdValue, err := GetTotalCoinsUsdValueAtHeight(
						client,
						preLiquidationHeight,
						sdk.NewCoins(initialLot),
						types.PriceType_Spot,
					)
					if err != nil {
						return err
					}

					if initialLot.Amount.LT(auctionData.AmountPurchased.Amount) {
						return fmt.Errorf(
							"auction %d: amount purchased (%s) is greater than initial lot (%s)",
							auctionData.ID, auctionData.AmountPurchased, initialLot,
						)
					}

					amountReturned := initialLot.Sub(auctionData.AmountPurchased)

					// Get total USD value after liquidation
					blocksIn1Hour := float64(hourSeconds) / float64(approxBlockSeconds)
					height4HoursAfter := auctionData.EndHeight + int64(4*blocksIn1Hour)

					// NOTE: fix for auctions within 4 hours of last block
					if height4HoursAfter > 1803249 {
						height4HoursAfter = 1803249
					}

					afterUsdValue, err := GetTotalCoinsUsdValueAtHeight(
						client,
						height4HoursAfter,
						sdk.NewCoins(amountReturned),
						types.PriceType_Spot,
					)
					if err != nil {
						return err
					}

					// (Before - After) / Before
					percentLossAmount := sdk.NewDecFromInt(initialLot.Amount).
						Sub(sdk.NewDecFromInt(amountReturned.Amount)).
						Quo(sdk.NewDecFromInt(initialLot.Amount))

					// (Before - After) / Before
					percentLossUsdValue := beforeUsdValue.
						Sub(afterUsdValue).
						Quo(beforeUsdValue)

					proceedsChan <- types.AuctionProceeds{
						BaseAuctionProceeds: auctionData,
						InitialLot:          initialLot,
						USDValueBefore:      beforeUsdValue,
						USDValueAfter:       afterUsdValue,
						AmountReturned:      amountReturned,
						PercentLossAmount:   percentLossAmount,
						PercentLossUSDValue: percentLossUsdValue,
					}

					return nil
				})
			}(auctionData)
		}

		bar.Finish()
	}()

	i := 0
	for aucProceed := range proceedsChan {
		dataMap[aucProceed.ID] = aucProceed

		i += 1
		if i == len(clearingData) {
			break
		}
	}

	err := g.Wait()
	return dataMap, err
}

func fetchLiquidationData(
	ctx context.Context,
	logger log.Logger,
	client Client,
	auctionProceeds types.BaseAuctionProceeds,
) (sdk.Coin, int64) {
	failedAttempts := 0

	for {
		switch auctionProceeds.SourceModule {
		case "hard":
			amount, height, err := GetAuctionSourceHARD(ctx, client, auctionProceeds.ID)
			if err != nil {
				// Print and retry
				sleepDuration := getBackoffDuration(failedAttempts)
				logger.Error(
					"Error fetching auction source HARD deposit",
					"auction ID", auctionProceeds.ID,
					"err", err,
					"delaySeconds", sleepDuration.String(),
				)

				time.Sleep(sleepDuration)
				failedAttempts += 1

				continue
			}

			return amount, height
		case "cdp", "liquidator":
			// This is separate since CDP liquidations are mostly in BeginBlocker
			amount, height, err := GetAuctionStartLotCDP(ctx, client, auctionProceeds.ID)
			if err != nil {
				sleepDuration := getBackoffDuration(failedAttempts)
				logger.Error(
					"Error fetching auction source CDP deposit",
					"auction ID", auctionProceeds.ID,
					"err", err,
					"delaySeconds", sleepDuration.String(),
				)

				time.Sleep(sleepDuration)
				failedAttempts += 1
				continue
			}

			return amount, height
		default:
			panic(fmt.Sprintf("Unhandled auction source module: %s", auctionProceeds.SourceModule))
		}
	}
}

// func GetAuctionAtHeight(client Client, id uint64, height int64) (auctiontypes.Auction, error) {
// 	ctx := ctxAtHeight(height)
// 	res, err := fetchAuction(client, ctx, &auctiontypes.QueryAuctionRequest{AuctionId: id})
// 	if err != nil {
// 		return &auctiontypes.CollateralAuction{}, err
// 	}
// 	var auc auctiontypes.Auction
// 	client.cdc.UnpackAny(res.Auction, &auc)
// 	return auc, nil
// }

// func fetchAuction(client Client, ctx context.Context, req *auctiontypes.QueryAuctionRequest) (*auctiontypes.QueryAuctionResponse, error) {
// 	for {
// 		res, err := client.Auction.Auction(ctx, req)
// 		if err != nil {
// 			continue
// 		}
// 		return res, nil
// 	}
// }

// fetchAuction peels pairs off then queries them in an endless loop
func fetchAuction(
	logger log.Logger,
	client Client,
	pairs <-chan auctionPair,
	results chan<- auctionResponse,
) {
	for pair := range pairs {
		failedAttempts := 0

		for {
			auc, err := client.GetAuction(pair.height, pair.id)

			// TODO: handle ErrAuctionNotFound
			if err != nil {
				sleepDuration := getBackoffDuration(failedAttempts)
				logger.Error(
					"Error fetching auction, retrying...",
					"auctionID", pair.id, "height",
					pair.height, "error", err,
					"delay", sleepDuration.String(),
				)

				failedAttempts += 1
				time.Sleep(sleepDuration)
				continue
			}

			if auc == nil {
				panic(fmt.Sprintf(
					"Auction %d was nil at height %d, meaning it was not found",
					pair.id, pair.height,
				))
			}

			results <- auctionResponse{
				Auction:   auc,
				EndHeight: pair.height,
			}
			break
		}
	}
}

func getBackoffDuration(attempt int) time.Duration {
	return time.Duration(math.Pow(2, float64(attempt))) * time.Second
}

// func SearchAuctionEnd(client Client, ctx context.Context, id ) (auctiontypes.Auction, error) {

// }

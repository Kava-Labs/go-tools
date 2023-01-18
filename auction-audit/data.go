package main

import (
	"container/heap"
	"context"
	"fmt"
	"math"
	"strconv"
	"time"

	"github.com/schollz/progressbar/v3"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/kava-labs/go-tools/auction-audit/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
)

const (
	concurrency        = 2
	hourSeconds        = 3600
	approxBlockSeconds = 6
)

// func GetInboundTransfers(client GrpcClient, start, end int64) (map[*sdk.AccAddress]sdk.Coins, error) {
// 	heights := make(chan int64)
// 	rawOutput := make(chan)
// }

func GetAuctionEndData(
	logger log.Logger,
	client GrpcClient,
	start, end int64,
) (types.AuctionIDToHeightMap, error) {
	// communication setup: heights -> worker pool -> raw ouput -> buffer -> sorted output
	heights := make(chan int64)
	rawOutput := make(chan types.BlockData)
	sortedOutput := make(chan types.BlockData)

	// buffer output and order
	go func() {
	}()
	// spawn pool of workers
	for i := 0; i < concurrency; i++ {
		go fetchBlock(logger, client, heights, rawOutput)
	}
	// write heights to input channel
	go func() {
		for height := start; height <= end; height++ {
			heights <- height
		}
	}()
	// run routine for collecting & sorting blocks
	go sortBlocks(rawOutput, sortedOutput, start)

	// auction ID -> block height
	aucMap := make(types.AuctionIDToHeightMap)

	bar := progressbar.Default(end - start)
	for output := range sortedOutput {
		bar.Add(1)
		bar.Describe(fmt.Sprintf("Processing block %d", output.Block.Header.Height))

		for i, txBytes := range output.Block.Data.Txs {
			if output.BlockResult.TxsResults[i].Code != 0 {
				// Skip failed txs, TxResults have the same index as Txs
				continue
			}

			tx, err := client.Decoder.TxDecoder()(txBytes)
			if err != nil {
				return types.AuctionIDToHeightMap{}, fmt.Errorf("failed to decode block tx bytes: %w", err)
			}

			msgs := tx.GetMsgs()
			for _, msg := range msgs {
				switch msg := msg.(type) {
				case *auctiontypes.MsgPlaceBid:
					aucMap[msg.AuctionId] = output.Block.Header.Height
				}
			}
		}

		if output.Block.Header.Height == end {
			break
		}
	}
	bar.Finish()

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
	client GrpcClient,
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

		col, ok := res.Auction.(*auctiontypes.CollateralAuction)
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
	client GrpcClient,
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
	client GrpcClient,
	auctionProceeds types.BaseAuctionProceeds,
) (sdk.Coin, int64) {
	for {
		switch auctionProceeds.SourceModule {
		case "hard":
			amount, height, err := GetAuctionSourceHARD(ctx, client, auctionProceeds.ID)
			if err != nil {
				// Print and retry
				logger.Error("Error fetching auction source HARD deposit", "err", err)
				continue
			}

			return amount, height
		case "cdp", "liquidator":
			// This is separate since CDP liquidations are mostly in BeginBlocker
			amount, height, err := GetAuctionStartLotCDP(ctx, client, auctionProceeds.ID)
			if err != nil {
				logger.Error("Error fetching auction source CDP deposit", "err", err)
				continue
			}

			return amount, height
		default:
			panic(fmt.Sprintf("Unhandled auction source module: %s", auctionProceeds.SourceModule))
		}
	}
}

// func GetAuctionAtHeight(client GrpcClient, id uint64, height int64) (auctiontypes.Auction, error) {
// 	ctx := ctxAtHeight(height)
// 	res, err := fetchAuction(client, ctx, &auctiontypes.QueryAuctionRequest{AuctionId: id})
// 	if err != nil {
// 		return &auctiontypes.CollateralAuction{}, err
// 	}
// 	var auc auctiontypes.Auction
// 	client.cdc.UnpackAny(res.Auction, &auc)
// 	return auc, nil
// }

// func fetchAuction(client GrpcClient, ctx context.Context, req *auctiontypes.QueryAuctionRequest) (*auctiontypes.QueryAuctionResponse, error) {
// 	for {
// 		res, err := client.Auction.Auction(ctx, req)
// 		if err != nil {
// 			continue
// 		}
// 		return res, nil
// 	}
// }

func ctxAtHeight(height int64) context.Context {
	heightStr := strconv.FormatInt(height, 10)
	return metadata.AppendToOutgoingContext(context.Background(), grpctypes.GRPCBlockHeightHeader, heightStr)
}

// fetchBlock never gives up and keeps trying until it gets the block
func fetchBlock(
	logger log.Logger,
	client GrpcClient,
	heights <-chan int64,
	blockResults chan<- types.BlockData,
) {
	for height := range heights {
		failedAttempts := 0

		for {
			block, err1 := client.Tm.GetBlockByHeight(
				context.Background(),
				&tmservice.GetBlockByHeightRequest{
					Height: height,
				},
			)

			blockResult, err2 := client.Tendermint.BlockResults(
				context.Background(),
				&height,
			)

			if err1 != nil {
				sleepSeconds := math.Pow(2, float64(failedAttempts))
				logger.Error(
					"Error fetching block, retrying...",
					"height", height,
					"GetBlockByHeight error", err1,
					"BlockResults error", err2,
					"delaySeconds", sleepSeconds,
				)

				time.Sleep(time.Duration(sleepSeconds) * time.Second)
				failedAttempts += 1
				continue
			}

			blockResults <- types.BlockData{
				Block:       block.Block,
				BlockResult: blockResult,
			}
			break
		}
	}
}

// fetchAuction peels pairs off then queries them in an endless loop
func fetchAuction(
	logger log.Logger,
	client GrpcClient,
	pairs <-chan auctionPair,
	results chan<- auctionResponse,
) {
	for pair := range pairs {
		nilCount := 0

		ctx := ctxAtHeight(pair.height)
		for {
			res, err := client.Auction.Auction(ctx, &auctiontypes.QueryAuctionRequest{
				AuctionId: pair.id,
			})
			if err != nil {
				logger.Error(
					"Error fetching auction, retrying...",
					"auctionID", pair.id, "height", pair.height, "error", err,
				)
				continue
			}

			var auc auctiontypes.Auction
			if err := client.cdc.UnpackAny(res.Auction, &auc); err != nil {
				panic(fmt.Sprintf("Error unpacking auction, %s", err))
			}

			if auc == nil {
				nilCount += 1
				if nilCount > 5 {
					panic(fmt.Sprintf("Auction %d was nil 5 times in a row", pair.id))
				}

				sleepSeconds := math.Pow(2, float64(nilCount))
				logger.Error(
					"Auction is nil, retrying...",
					"auctionID", pair.id,
					"height", pair.height,
					"delaySeconds", sleepSeconds,
				)
				time.Sleep(time.Duration(sleepSeconds) * time.Second)
				continue
			}

			results <- auctionResponse{
				Auction:   auc,
				EndHeight: pair.height,
			}
			break
		}
	}
}

// buffers and ouputs sorted blocks using a heap
func sortBlocks(
	unsorted <-chan types.BlockData,
	sorted chan<- types.BlockData,
	start int64,
) {
	queue := &types.BlockHeap{}
	previousHeight := start - 1

	for result := range unsorted {
		heap.Push(queue, result)

		for queue.Len() > 0 {
			minHeight := (*queue)[0].Block.Header.Height

			if minHeight == previousHeight+1 {
				result := heap.Pop(queue).(types.BlockData)

				sorted <- result

				previousHeight = result.Block.Header.Height
				continue
			}

			break
		}
	}
}

// func SearchAuctionEnd(client GrpcClient, ctx context.Context, id ) (auctiontypes.Auction, error) {

// }

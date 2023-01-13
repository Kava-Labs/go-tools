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
	concurrency = 2
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
	rawOutput := make(chan *tmservice.GetBlockByHeightResponse)
	sortedOutput := make(chan *tmservice.GetBlockByHeightResponse)

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

		for _, txBytes := range output.Block.Data.Txs {
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

func GetAuctionClearingData(
	logger log.Logger,
	client GrpcClient,
	endMap map[uint64]int64,
) (types.BaseAuctionProceedsMap, error) {
	// Return if empty, otherwise the loop over rawOutput will hang forever:
	// rawOutput will never have any values to read, so the loop will never start.
	if len(endMap) == 0 {
		return types.BaseAuctionProceedsMap{}, nil
	}

	rawOutput := make(chan *auctiontypes.QueryAuctionResponse)
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

	i := 1
	for res := range rawOutput {
		var auc auctiontypes.Auction
		client.cdc.UnpackAny(res.Auction, &auc)

		col, ok := auc.(*auctiontypes.CollateralAuction)
		if ok {
			ap, found := clearingMap[col.GetID()]

			if found {
				ap.AmountPurchased = ap.AmountPurchased.Add(col.GetLot())
				ap.AmountPaid = ap.AmountPaid.Add(col.GetBid())
				ap.InitialLot = sdk.NewCoin(col.GetLot().Denom, col.GetLotReturns().Weights[0])
				ap.LiquidatedAccount = col.GetLotReturns().Addresses[0].String()
				ap.WinningBidder = col.GetBidder().String()

				clearingMap[col.GetID()] = ap
			} else {
				clearingMap[col.GetID()] = types.BaseAuctionProceeds{
					ID:                col.GetID(),
					AmountPurchased:   col.GetLot(),
					AmountPaid:        col.GetBid(),
					InitialLot:        sdk.NewCoin(col.GetLot().Denom, col.GetLotReturns().Weights[0]),
					LiquidatedAccount: col.GetLotReturns().Addresses[0].String(),
					WinningBidder:     col.GetBidder().String(),
					SourceModule:      auc.GetInitiator(),
				}
			}
		}

		if i == len(endMap) {
			break
		}
		i++
	}

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

	for _, auctionData := range clearingData {
		func(auctionData types.BaseAuctionProceeds) {
			g.Go(func() error {
				// Coins from cdp or hard deposit 1 block before liquidation
				preLiquidationAmount, preLiquidationHeight := fetchAuctionSourceAmount(ctx, logger, client, auctionData)

				// Get USD value
				beforeUsdValue, err := GetTotalCoinsUsdValueAtHeight(client, preLiquidationHeight, preLiquidationAmount, types.Spot)
				if err != nil {
					return err
				}

				proceedsChan <- types.AuctionProceeds{
					BaseAuctionProceeds: auctionData,
					UsdValueBefore:      beforeUsdValue,
				}

				return nil
			})
		}(auctionData)
	}

	i := 0
	for aucProceed := range proceedsChan {
		dataMap[aucProceed.ID] = aucProceed

		logger.Debug(
			"processed full auction proceed",
			"i", i,
			"auctionID", aucProceed.ID,
			"value", aucProceed.UsdValueBefore.String(),
		)

		i += 1
		if i == len(clearingData) {
			break
		}
	}

	err := g.Wait()
	return dataMap, err
}

func fetchAuctionSourceAmount(
	ctx context.Context,
	logger log.Logger,
	client GrpcClient,
	auctionProceeds types.BaseAuctionProceeds,
) (sdk.Coins, int64) {
	for {
		switch auctionProceeds.SourceModule {
		case "hard":
			hardDeposit, height, err := GetAuctionSourceHARD(ctx, client, auctionProceeds.ID)
			if err != nil {
				// Print and retry
				logger.Error("Error fetching auction source HARD deposit", "err", err)
				continue
			}

			return hardDeposit.Amount, height
		case "cdp", "liquidator":
			cdpDeposit, height, err := GetAuctionSourceCDP(ctx, client, auctionProceeds.ID)
			if err != nil {
				logger.Error("Error fetching auction source CDP deposit", "err", err)
				continue
			}

			return sdk.NewCoins(cdpDeposit.Collateral), height
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
	blockResults chan<- *tmservice.GetBlockByHeightResponse,
) {
	for height := range heights {
		failedAttempts := 0

		for {
			result, err := client.Tm.GetBlockByHeight(
				context.Background(),
				&tmservice.GetBlockByHeightRequest{
					Height: height,
				},
			)

			if err != nil {
				sleepSeconds := math.Pow(2, float64(failedAttempts))
				logger.Error(
					"Error fetching block, retrying...",
					"height", height,
					"error", err,
					"delaySeconds", sleepSeconds,
				)

				time.Sleep(time.Duration(sleepSeconds) * time.Second)
				failedAttempts += 1
				continue
			}

			blockResults <- result
			break
		}
	}
}

// fetchAuction peels pairs off then queries them in an endless loop
func fetchAuction(
	logger log.Logger,
	client GrpcClient,
	pairs <-chan auctionPair,
	results chan<- *auctiontypes.QueryAuctionResponse,
) {
	for pair := range pairs {
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

			results <- res
			break
		}
	}
}

// buffers and ouputs sorted blocks using a heap
func sortBlocks(
	unsorted <-chan *tmservice.GetBlockByHeightResponse,
	sorted chan<- *tmservice.GetBlockByHeightResponse,
	start int64,
) {
	queue := &types.BlockHeap{}
	previousHeight := start - 1

	for result := range unsorted {
		heap.Push(queue, result)

		for queue.Len() > 0 {
			minHeight := (*queue)[0].Block.Header.Height

			if minHeight == previousHeight+1 {
				result := heap.Pop(queue).(*tmservice.GetBlockByHeightResponse)

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

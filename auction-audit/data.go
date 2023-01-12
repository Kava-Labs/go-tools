package main

import (
	"container/heap"
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/kava-labs/go-tools/auction-audit/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc/metadata"
)

const (
	concurrency = 100
)

// func GetInboundTransfers(client GrpcClient, start, end int64) (map[*sdk.AccAddress]sdk.Coins, error) {
// 	heights := make(chan int64)
// 	rawOutput := make(chan)
// }

func GetAuctionEndData(
	client GrpcClient,
	start, end int64,
	bidder sdk.AccAddress,
) (types.AuctionIDToHeightMap, map[string]sdk.Coins, error) {
	// communication setup: heights -> worker pool -> raw ouput -> buffer -> sorted output
	heights := make(chan int64)
	rawOutput := make(chan *tmservice.GetBlockByHeightResponse)
	sortedOutput := make(chan *tmservice.GetBlockByHeightResponse)

	// buffer output and order
	go func() {
	}()
	// spawn pool of workers
	for i := 0; i < concurrency; i++ {
		go fetchBlock(client, heights, rawOutput)
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
	// bidder address -> amount
	transferMap := make(map[string]sdk.Coins)

	for output := range sortedOutput {
		for _, txBytes := range output.Block.Data.Txs {
			tx, err := client.Decoder.TxDecoder()(txBytes)
			if err != nil {
				return types.AuctionIDToHeightMap{}, map[string]sdk.Coins{}, err
			}

			msgs := tx.GetMsgs()
			for _, msg := range msgs {
				switch msg := msg.(type) {
				case *auctiontypes.MsgPlaceBid:
					id := msg.AuctionId
					aucMap[id] = output.Block.Header.Height
				case *banktypes.MsgSend:
					if msg.ToAddress == bidder.String() {
						// Default empty coins if not found
						sendAmount := transferMap[msg.FromAddress]

						transferMap[msg.FromAddress] = sendAmount.Add(msg.Amount...)
					}
				}
			}
		}

		if output.Block.Header.Height == end {
			break
		}
	}

	return aucMap, transferMap, nil
}

type auctionPair struct {
	id     uint64
	height int64
}

func GetAuctionClearingData(
	client GrpcClient,
	endMap map[uint64]int64,
	bidder sdk.AccAddress,
) (types.BaseAuctionProceedsMap, error) {
	rawOutput := make(chan *auctiontypes.QueryAuctionResponse)
	pairs := make(chan auctionPair)

	clearingMap := make(types.BaseAuctionProceedsMap)

	// buffer output and order
	go func() {
	}()
	// spawn pool of workers
	for i := 0; i < concurrency; i++ {
		go fetchAuction(client, pairs, rawOutput)
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

		// TODO: Get prices after auction ends

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
	client GrpcClient,
	clearingData types.BaseAuctionProceedsMap,
) (types.AuctionProceedsMap, error) {
	g, ctx := errgroup.WithContext(ctx)
	g.SetLimit(concurrency)

	proceedsChan := make(chan types.AuctionProceeds)
	dataMap := make(types.AuctionProceedsMap)

	for _, auctionData := range clearingData {
		func(auctionData types.BaseAuctionProceeds) {
			g.Go(func() error {
				// Coins from cdp or hard deposit 1 block before liquidation
				preLiquidationAmount, preLiquidationHeight := fetchAuctionSourceAmount(ctx, client, auctionData)

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
	client GrpcClient,
	auctionProceeds types.BaseAuctionProceeds,
) (sdk.Coins, int64) {
	for {
		switch auctionProceeds.SourceModule {
		case "hard":
			hardDeposit, height, err := GetAuctionSourceHARD(ctx, client, auctionProceeds.ID)
			if err != nil {
				// Print and retry
				fmt.Fprintf(os.Stderr, "Error fetching auction source HARD deposit: %s", err)
				continue
			}

			return hardDeposit.Amount, height
		case "cdp":
			cdpDeposit, height, err := GetAuctionSourceCDP(ctx, client, auctionProceeds.ID)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error fetching auction source CDP deposit: %s", err)
				continue
			}

			return sdk.NewCoins(cdpDeposit.Collateral), height
		default:
			fmt.Fprintf(os.Stderr, "Unhandled auction source module: %s", auctionProceeds.SourceModule)
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
func fetchBlock(client GrpcClient, heights <-chan int64, blockResults chan<- *tmservice.GetBlockByHeightResponse) {
	for height := range heights {
		for {
			result, err := client.Tm.GetBlockByHeight(
				context.Background(),
				&tmservice.GetBlockByHeightRequest{
					Height: height,
				},
			)

			if err != nil {
				continue
			}

			blockResults <- result
			break
		}
	}
}

// fetchAuction peels pairs off then queries them in an endless loop
func fetchAuction(client GrpcClient, pairs <-chan auctionPair, results chan<- *auctiontypes.QueryAuctionResponse) {
	for pair := range pairs {
		ctx := ctxAtHeight(pair.height)
		for {
			res, err := client.Auction.Auction(ctx, &auctiontypes.QueryAuctionRequest{AuctionId: pair.id})
			if err != nil {
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

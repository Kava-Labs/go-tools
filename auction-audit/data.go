package main

import (
	"container/heap"
	"context"
	"fmt"
	"strconv"

	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	sdk "github.com/cosmos/cosmos-sdk/types"
	grpctypes "github.com/cosmos/cosmos-sdk/types/grpc"
	"github.com/cosmos/cosmos-sdk/types/tx"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	"google.golang.org/grpc/metadata"
)

const (
	concurrency = 100
)

// func GetInboundTransfers(client GrpcClient, start, end int64) (map[*sdk.AccAddress]sdk.Coins, error) {
// 	heights := make(chan int64)
// 	rawOutput := make(chan)
// }

type CDPAuctionPair struct {
	CdpId     sdk.Int
	AuctionId sdk.Int
}

func getCdpAuctionPairFromTxResponse(txResponse *sdk.TxResponse) (CDPAuctionPair, error) {
	var cdpId sdk.Int
	var auctionId sdk.Int
	cdpIdFound := false
	auctionIdFound := false

	for _, log := range txResponse.Logs {
		for _, event := range log.Events {
			if event.Type != cdptypes.EventTypeCdpLiquidation &&
				event.Type != auctiontypes.EventTypeAuctionStart {
				continue
			}

			for _, attr := range event.Attributes {
				if attr.Key == cdptypes.AttributeKeyCdpID {
					id, ok := sdk.NewIntFromString(attr.Value)
					if !ok {
						return CDPAuctionPair{}, fmt.Errorf("failed to parse cdp id: %s", attr.Value)
					}

					cdpId = id
					cdpIdFound = true
				}

				if attr.Key == auctiontypes.AttributeKeyAuctionID {
					id, ok := sdk.NewIntFromString(attr.Value)
					if !ok {
						return CDPAuctionPair{}, fmt.Errorf("failed to parse auction id: %s", attr.Value)
					}

					auctionId = id
					auctionIdFound = true
				}
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
	}, nil
}

func GetAuctionSourceCDP(
	ctx context.Context,
	client GrpcClient,
	height int64,
	auctionID int64,
) (cdptypes.CDPResponse, error) {
	res, err := client.Tx.GetTxsEvent(
		ctx,
		&tx.GetTxsEventRequest{
			Events: []string{
				// Query service concats multiple events with AND
				"cdp_liquidation.module='cdp'",
				fmt.Sprintf(
					"tx.height=%d",
					height,
				),
			},
		},
	)
	if err != nil {
		return cdptypes.CDPResponse{}, err
	}

	var pairs []CDPAuctionPair

	// Get corresponding CDP from liquidate event
	for _, tsRes := range res.TxResponses {
		// There can be multiple liquidations in a single block
		pair, err := getCdpAuctionPairFromTxResponse(tsRes)
		if err != nil {
			// There must be a matching event in every TxResponse, as we are
			// querying for the matching event.
			return cdptypes.CDPResponse{}, err
		}

		pairs = append(pairs, pair)
	}

	// Find matching CDP
	var matchingCdpId sdk.Int
	found := false
	for _, pair := range pairs {
		if pair.AuctionId.Equal(sdk.NewInt(auctionID)) {
			matchingCdpId = pair.CdpId
			found = true
		}
	}

	if !found {
		return cdptypes.CDPResponse{}, fmt.Errorf("failed to find matching CDP for auction ID %d", auctionID)
	}

	// Query CDP at height 1 before liquidation, as it is deleted when liquidated
	queryCtx := ctxAtHeight(height - 1)

	// Fetch CDP to determine original value prior to liquidation
	cdpRes, err := client.CDP.Cdps(queryCtx, &cdptypes.QueryCdpsRequest{
		ID: uint64(matchingCdpId.Int64()),
	})

	if err != nil {
		return cdptypes.CDPResponse{}, err
	}

	if len(cdpRes.Cdps) == 0 {
		return cdptypes.CDPResponse{}, fmt.Errorf("cdp %d was not found at block %d", matchingCdpId.Int64(), height-1)
	}

	return cdpRes.Cdps[0], nil
}

func GetAuctionEndData(
	client GrpcClient,
	start, end int64,
	bidder sdk.AccAddress,
) (map[uint64]int64, map[string]sdk.Coins, error) {
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
	aucMap := make(map[uint64]int64)
	// bidder address -> amount
	transferMap := make(map[string]sdk.Coins)

	for output := range sortedOutput {
		for _, txBytes := range output.Block.Data.Txs {
			tx, err := client.Decoder.TxDecoder()(txBytes)
			if err != nil {
				return map[uint64]int64{}, map[string]sdk.Coins{}, err
			}

			msgs := tx.GetMsgs()
			for _, msg := range msgs {
				bidMsg, ok := msg.(*auctiontypes.MsgPlaceBid)
				if ok {
					id := bidMsg.AuctionId
					aucMap[id] = output.Block.Header.Height
				}
				sendMsg, ok := msg.(*banktypes.MsgSend)
				if ok {
					if sendMsg.ToAddress == bidder.String() {
						sendAmount, found := transferMap[sendMsg.FromAddress]
						if found {
							transferMap[sendMsg.FromAddress] = sendAmount.Add(sendMsg.Amount...)
						} else {
							transferMap[sendMsg.FromAddress] = sendMsg.Amount
						}
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
) (map[uint64]auctionProceeds, error) {
	rawOutput := make(chan *auctiontypes.QueryAuctionResponse)
	pairs := make(chan auctionPair)

	clearingMap := make(map[uint64]auctionProceeds)

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

		// Get prices after auction ends

		col, ok := auc.(*auctiontypes.CollateralAuction)
		if ok {
			ap, ok2 := clearingMap[col.GetID()]
			if ok2 {
				ap.AmountPurchased = ap.AmountPurchased.Add(col.GetLot())
				ap.AmountPaid = ap.AmountPaid.Add(col.GetBid())
				ap.InitialLot = sdk.Coin{Denom: col.GetLot().Denom, Amount: col.GetLotReturns().Weights[0]}
				ap.LiquidatedAccount = col.GetLotReturns().Addresses[0].String()
				ap.WinningBidder = col.GetBidder().String()
				clearingMap[col.GetID()] = ap
			} else {
				clearingMap[col.GetID()] = auctionProceeds{
					AmountPurchased:   col.GetLot(),
					AmountPaid:        col.GetBid(),
					InitialLot:        sdk.Coin{Denom: col.GetLot().Denom, Amount: col.GetLotReturns().Weights[0]},
					LiquidatedAccount: col.GetLotReturns().Addresses[0].String(),
					WinningBidder:     col.GetBidder().String(),
					// TODO: New fields
					SourceModule: auc.GetInitiator(),
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
	queue := &BlockHeap{}
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

type auctionProceeds struct {
	AmountPurchased   sdk.Coin
	AmountPaid        sdk.Coin
	InitialLot        sdk.Coin
	LiquidatedAccount string
	WinningBidder     string
	ValueBeforeUsd    sdk.Int
	ValueAfterUsd     sdk.Int
	PercentLoss       sdk.Dec
	SourceModule      string
}

type BlockHeap []*tmservice.GetBlockByHeightResponse

func (h BlockHeap) Len() int { return len(h) }
func (h BlockHeap) Less(i, j int) bool {
	return h[i].Block.Header.Height < h[j].Block.Header.Height
}
func (h BlockHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }

func (h *BlockHeap) Push(x interface{}) {
	*h = append(*h, x.(*tmservice.GetBlockByHeightResponse))
}

func (h *BlockHeap) Pop() interface{} {
	old := *h
	n := len(old)
	x := old[n-1]
	*h = old[0 : n-1]
	return x
}

// func SearchAuctionEnd(client GrpcClient, ctx context.Context, id ) (auctiontypes.Auction, error) {

// }

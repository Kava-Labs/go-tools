package auctions

import (
	"context"
	"fmt"
	"github.com/cosmos/cosmos-sdk/client/grpc/tmservice"
	"github.com/cosmos/cosmos-sdk/codec"
	query "github.com/cosmos/cosmos-sdk/types/query"
	kavagrpc "github.com/kava-labs/kava/client/grpc"
	kavagrpcutil "github.com/kava-labs/kava/client/grpc/util"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
)

const (
	DefaultPageLimit = 1000
)

// InfoResponse defines the ID and latest height for a specific chain
type InfoResponse struct {
	ChainId      string `json:"chain_id" yaml:"chain_id"`
	LatestHeight int64  `json:"latest_height" yaml:"latest_height"`
}

// AuctionClient defines the expected client interface for interacting with auctions
type AuctionClient interface {
	GetInfo() (*InfoResponse, error)
	GetPrices(height int64) (pricefeedtypes.CurrentPriceResponses, error)
	GetAuctions(height int64) ([]auctiontypes.Auction, error)
	GetCollateralParams(height int64) (cdptypes.CollateralParams, error)
	GetMoneyMarkets(height int64) (hardtypes.MoneyMarkets, error)
}

// GrpcAuctionClient defines a client for interacting with auctions via rpc
type GrpcAuctionClient struct {
	grpcClient *kavagrpc.KavaGrpcClient
	util       *kavagrpcutil.Util
	cdc        codec.Codec
	PageLimit  uint64
}

var _ AuctionClient = (*GrpcAuctionClient)(nil)

// NewGrpcAuctionClient returns a new GrpcAuctionClient
func NewGrpcAuctionClient(
	grpcClient *kavagrpc.KavaGrpcClient,
	cdc codec.Codec,
) *GrpcAuctionClient {
	return &GrpcAuctionClient{
		grpcClient: grpcClient,
		cdc:        cdc,
		util:       kavagrpcutil.NewUtil(nil), // we need util to create context for particular height
		PageLimit:  DefaultPageLimit,
	}
}

// GetInfo returns the current chain info
func (c *GrpcAuctionClient) GetInfo() (*InfoResponse, error) {
	resultLatestBlock, err := c.grpcClient.Query.Tm.GetLatestBlock(context.Background(), &tmservice.GetLatestBlockRequest{})
	if err != nil {
		return nil, err
	}

	resultNodeInfo, err := c.grpcClient.Query.Tm.GetNodeInfo(context.Background(), &tmservice.GetNodeInfoRequest{})
	if err != nil {
		return nil, err
	}

	return &InfoResponse{
		ChainId:      resultNodeInfo.GetDefaultNodeInfo().Network,
		LatestHeight: resultLatestBlock.GetBlock().Header.Height,
	}, nil
}

// GetPrices gets the current prices for markets for provided height
func (c *GrpcAuctionClient) GetPrices(height int64) (pricefeedtypes.CurrentPriceResponses, error) {
	heightCtx := c.util.CtxAtHeight(height)
	prices, err := c.grpcClient.Query.Pricefeed.Prices(
		heightCtx,
		&pricefeedtypes.QueryPricesRequest{},
	)
	if err != nil {
		return nil, err
	}

	return prices.Prices, nil
}

// GetCollateralParams gets an array of collateral params for each collateral type for provided height
func (c *GrpcAuctionClient) GetCollateralParams(height int64) (cdptypes.CollateralParams, error) {
	heightCtx := c.util.CtxAtHeight(height)
	params, err := c.grpcClient.Query.Cdp.Params(heightCtx, &cdptypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return params.Params.CollateralParams, nil
}

// GetMoneyMarkets gets an array of money markets for each asset for provided height
func (c *GrpcAuctionClient) GetMoneyMarkets(height int64) (hardtypes.MoneyMarkets, error) {
	heightCtx := c.util.CtxAtHeight(height)
	params, err := c.grpcClient.Query.Hard.Params(heightCtx, &hardtypes.QueryParamsRequest{})
	if err != nil {
		return nil, err
	}

	return params.Params.MoneyMarkets, nil
}

// GetAuctions gets all the currently running auctions for provided height
func (c *GrpcAuctionClient) GetAuctions(height int64) ([]auctiontypes.Auction, error) {
	var (
		auctions []auctiontypes.Auction
		key      []byte
	)

	for {
		heightCtx := c.util.CtxAtHeight(height)
		auctionsResponse, err := c.grpcClient.Query.Auction.Auctions(heightCtx, &auctiontypes.QueryAuctionsRequest{
			Pagination: &query.PageRequest{
				Key:   key,
				Limit: c.PageLimit,
			},
		})
		if err != nil {
			return nil, err
		}

		for _, anyAuction := range auctionsResponse.GetAuctions() {
			var auction auctiontypes.Auction
			if err = c.cdc.UnpackAny(anyAuction, &auction); err != nil {
				return nil, fmt.Errorf("failed to unpack auction: %w", err)
			}

			auctions = append(auctions, auction)
		}

		if auctionsResponse.Pagination == nil || auctionsResponse.Pagination.NextKey == nil {
			break
		}
	}

	return auctions, nil
}

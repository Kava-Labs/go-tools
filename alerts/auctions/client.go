package auctions

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"github.com/tendermint/tendermint/libs/bytes"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
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
	GetPrices(height int64) (pricefeedtypes.CurrentPrices, error)
	GetAuctions(height int64) ([]auctiontypes.Auction, error)
	GetMarkets(height int64) (cdptypes.CollateralParams, error)
	GetMoneyMarkets(height int64) (hardtypes.MoneyMarkets, error)
}

// RpcAuctionClient defines a client for interacting with auctions via rpc
type RpcAuctionClient struct {
	rpc       RpcClient
	cdc       codec.LegacyAmino
	PageLimit int
}

var _ AuctionClient = (*RpcAuctionClient)(nil)

// NewRpcAuctionClient returns a new RpcAuctionClient
func NewRpcAuctionClient(rpc RpcClient, cdc codec.LegacyAmino) *RpcAuctionClient {
	return &RpcAuctionClient{
		rpc:       rpc,
		cdc:       cdc,
		PageLimit: DefaultPageLimit,
	}
}

// GetInfo returns the current chain info
func (c *RpcAuctionClient) GetInfo() (*InfoResponse, error) {
	result, err := c.rpc.Status(context.Background())
	if err != nil {
		return nil, err
	}

	return &InfoResponse{
		ChainId:      result.NodeInfo.Network,
		LatestHeight: result.SyncInfo.LatestBlockHeight,
	}, nil
}

// GetPrices gets the current prices for markets
func (c *RpcAuctionClient) GetPrices(height int64) (pricefeedtypes.CurrentPrices, error) {
	path := fmt.Sprintf("custom/%s/%s", pricefeedtypes.QuerierRoute, pricefeedtypes.QueryPrices)

	data, err := c.abciQuery(path, bytes.HexBytes{}, height)
	if err != nil {
		return nil, err
	}

	var currentPrices pricefeedtypes.CurrentPrices
	err = c.cdc.UnmarshalJSON(data, &currentPrices)
	if err != nil {
		return nil, err
	}

	return currentPrices, nil
}

// GetMarkets gets an array of collateral params for each collateral type
func (c *RpcAuctionClient) GetMarkets(height int64) (cdptypes.CollateralParams, error) {
	path := fmt.Sprintf("custom/%s/%s", cdptypes.QuerierRoute, cdptypes.QueryGetParams)

	data, err := c.abciQuery(path, bytes.HexBytes{}, height)
	if err != nil {
		return nil, err
	}

	var params cdptypes.Params
	err = c.cdc.UnmarshalJSON(data, &params)
	if err != nil {
		return nil, err
	}

	return params.CollateralParams, nil
}

// GetMoneyMarkets gets an array of money markets for each asset
func (c *RpcAuctionClient) GetMoneyMarkets(height int64) (hardtypes.MoneyMarkets, error) {
	path := fmt.Sprintf("custom/%s/%s", hardtypes.QuerierRoute, hardtypes.QueryGetParams)
	data, err := c.abciQuery(path, bytes.HexBytes{}, height)
	if err != nil {
		return nil, err
	}

	var params hardtypes.Params
	err = c.cdc.UnmarshalJSON(data, &params)
	if err != nil {
		return nil, err
	}
	return params.MoneyMarkets, nil
}

// GetAuctions gets all the currently running auctions
func (c *RpcAuctionClient) GetAuctions(height int64) ([]auctiontypes.Auction, error) {
	path := fmt.Sprintf("custom/%s/%s", auctiontypes.QuerierRoute, auctiontypes.QueryGetAuctions)

	page := 1
	var auctions []auctiontypes.Auction

	for {
		params := auctiontypes.NewQueryAllAuctionParams(page, c.PageLimit, "", "", "", sdk.AccAddress{})
		bz, err := c.cdc.MarshalJSON(&params)

		if err != nil {
			return nil, err
		}

		data, err := c.abciQuery(path, bz, height)
		if err != nil {
			return nil, err
		}

		var pagedAuctions []auctiontypes.Auction
		err = c.cdc.UnmarshalJSON(data, &pagedAuctions)
		if err != nil {
			return nil, err
		}

		if len(pagedAuctions) > 0 {
			auctions = append(auctions, pagedAuctions...)
		}

		if len(pagedAuctions) < c.PageLimit {
			return auctions, nil
		}

		page++
	}
}

func (c *RpcAuctionClient) abciQuery(
	path string,
	data bytes.HexBytes,
	height int64) ([]byte, error) {
	opts := rpcclient.ABCIQueryOptions{Height: height, Prove: false}

	result, err := c.rpc.ABCIQueryWithOptions(context.Background(), path, data, opts)
	if err != nil {
		return []byte{}, err
	}

	resp := result.Response
	if !resp.IsOK() {
		return []byte{}, errors.New(resp.Log)
	}

	// TODO: why do we check length here?
	value := result.Response.GetValue()
	// TODO: untested logic case
	if len(value) == 0 {
		return []byte{}, nil
	}

	return value, nil
}

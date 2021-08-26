package swap

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/kava-labs/go-tools/alerts/rpc"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	swaptypes "github.com/kava-labs/kava/x/swap/types"
	"github.com/tendermint/tendermint/libs/bytes"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

// InfoResponse defines the ID and latest height for a specific chain
type InfoResponse struct {
	ChainId      string `json:"chain_id" yaml:"chain_id"`
	LatestHeight int64  `json:"latest_height" yaml:"latest_height"`
}

// SwapClient defines the expected client interface for interacting with swap
type SwapClient interface {
	GetInfo() (*InfoResponse, error)
	GetPrices(height int64) (pricefeedtypes.CurrentPrices, error)
	GetPools(height int64) (swaptypes.PoolStatsQueryResults, error)
	GetMarkets(height int64) (cdptypes.CollateralParams, error)
	GetMoneyMarkets(height int64) (hardtypes.MoneyMarkets, error)
}

// RpcSwapClient defines a client for interacting with auctions via rpc
type RpcSwapClient struct {
	rpc rpc.RpcClient
	cdc *codec.Codec
}

var _ SwapClient = (*RpcSwapClient)(nil)

// NewRpcSwapClient returns a new RpcSwapClient
func NewRpcSwapClient(rpc rpc.RpcClient, cdc *codec.Codec) *RpcSwapClient {
	return &RpcSwapClient{
		rpc: rpc,
		cdc: cdc,
	}
}

// GetInfo returns the current chain info
func (c *RpcSwapClient) GetInfo() (*InfoResponse, error) {
	result, err := c.rpc.Status()
	if err != nil {
		return nil, err
	}

	return &InfoResponse{
		ChainId:      result.NodeInfo.Network,
		LatestHeight: result.SyncInfo.LatestBlockHeight,
	}, nil
}

// GetPrices gets the current prices for markets
func (c *RpcSwapClient) GetPrices(height int64) (pricefeedtypes.CurrentPrices, error) {
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
func (c *RpcSwapClient) GetMarkets(height int64) (cdptypes.CollateralParams, error) {
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
func (c *RpcSwapClient) GetMoneyMarkets(height int64) (hardtypes.MoneyMarkets, error) {
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
func (c *RpcSwapClient) GetPools(height int64) (swaptypes.PoolStatsQueryResults, error) {
	path := fmt.Sprintf("custom/%s/%s", swaptypes.QuerierRoute, swaptypes.QueryGetPools)

	data, err := c.abciQuery(path, bytes.HexBytes{}, height)
	if err != nil {
		return nil, err
	}

	var pools swaptypes.PoolStatsQueryResults
	if err := c.cdc.UnmarshalJSON(data, &pools); err != nil {
		return nil, err
	}

	return pools, nil
}

func (c *RpcSwapClient) abciQuery(
	path string,
	data bytes.HexBytes,
	height int64) ([]byte, error) {
	opts := rpcclient.ABCIQueryOptions{Height: height, Prove: false}

	result, err := c.rpc.ABCIQueryWithOptions(path, data, opts)
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

package main

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	cdptypes "github.com/kava-labs/kava/x/cdp/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	swaptypes "github.com/kava-labs/kava/x/swap/types"
	"github.com/tendermint/tendermint/libs/bytes"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
)

const (
	DefaultPageLimit = 1000
)

type InfoResponse struct {
	ChainId      string `json:"chain_id" yaml:"chain_id"`
	LatestHeight int64  `json:"latest_height" yaml:"latest_height"`
}

type SwapClient interface {
	GetInfo() (*InfoResponse, error)
	GetPrices(height int64) (pricefeedtypes.CurrentPrices, error)
	GetPools(height int64) (swaptypes.PoolStatsQueryResults, error)
	GetMarkets(height int64) (cdptypes.CollateralParams, error)
}

type RpcSwapClient struct {
	rpc       RpcClient    `json:"rpc" yaml:"rpc"`
	cdc       *codec.Codec `json:"cdc" yaml:"cdc"`
	PageLimit int
}

var _ SwapClient = (*RpcSwapClient)(nil)

func NewRpcSwapClient(rpc RpcClient, cdc *codec.Codec) *RpcSwapClient {
	return &RpcSwapClient{
		rpc:       rpc,
		cdc:       cdc,
		PageLimit: DefaultPageLimit,
	}
}

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

func (c *RpcSwapClient) GetPools(height int64) (swaptypes.PoolStatsQueryResults, error) {
	path := fmt.Sprintf("custom/%s/%s", swaptypes.QuerierRoute, swaptypes.QueryGetPools)

	data, err := c.abciQuery(path, bytes.HexBytes{}, height)
	if err != nil {
		return nil, err
	}

	var params swaptypes.PoolStatsQueryResults
	err = c.cdc.UnmarshalJSON(data, &params)
	if err != nil {
		return nil, err
	}

	return params, nil
}

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

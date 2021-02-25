package main

import (
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	hardtypes "github.com/kava-labs/kava/x/hard/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"github.com/tendermint/tendermint/libs/bytes"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

const (
	DefaultPageLimit = 1000
)

type RpcClient interface {
	Status() (*ctypes.ResultStatus, error)
	ABCIQueryWithOptions(
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) (*ctypes.ResultABCIQuery, error)
}

type InfoResponse struct {
	ChainId      string
	LatestHeight int64
}

type LiquidationClient interface {
	GetInfo() (*InfoResponse, error)
	GetPrices(height int64) (pricefeedtypes.CurrentPrices, error)
	GetMarkets(height int64) (hardtypes.MoneyMarkets, error)
	GetBorrows(height int64) (hardtypes.Borrows, error)
	GetDeposits(height int64) (hardtypes.Deposits, error)
}

type RpcLiquidationClient struct {
	rpc       RpcClient
	cdc       *codec.Codec
	PageLimit int
}

var _ LiquidationClient = (*RpcLiquidationClient)(nil)

func NewRpcLiquidationClient(rpc RpcClient, cdc *codec.Codec) *RpcLiquidationClient {
	return &RpcLiquidationClient{
		rpc:       rpc,
		cdc:       cdc,
		PageLimit: DefaultPageLimit,
	}
}

func (c *RpcLiquidationClient) GetInfo() (*InfoResponse, error) {
	result, err := c.rpc.Status()
	if err != nil {
		return nil, err
	}

	return &InfoResponse{
		ChainId:      result.NodeInfo.Network,
		LatestHeight: result.SyncInfo.LatestBlockHeight,
	}, nil
}

func (c *RpcLiquidationClient) GetPrices(height int64) (pricefeedtypes.CurrentPrices, error) {
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

func (c *RpcLiquidationClient) GetMarkets(height int64) (hardtypes.MoneyMarkets, error) {
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

func (c *RpcLiquidationClient) GetBorrows(height int64) (hardtypes.Borrows, error) {
	path := fmt.Sprintf("custom/%s/%s", hardtypes.QuerierRoute, hardtypes.QueryGetBorrows)

	page := 1
	var borrows hardtypes.Borrows

	for {
		params := hardtypes.NewQueryBorrowsParams(page, c.PageLimit, sdk.AccAddress{}, "")

		bz, err := c.cdc.MarshalJSON(&params)
		if err != nil {
			return nil, err
		}

		data, err := c.abciQuery(path, bz, height)
		if err != nil {
			return nil, err
		}

		var pagedBorrows hardtypes.Borrows
		err = c.cdc.UnmarshalJSON(data, &pagedBorrows)
		if err != nil {
			return nil, err
		}

		if len(pagedBorrows) > 0 {
			borrows = append(borrows, pagedBorrows...)
		}

		if len(pagedBorrows) < c.PageLimit {
			return borrows, nil
		}

		page++
	}
}

func (c *RpcLiquidationClient) GetDeposits(height int64) (hardtypes.Deposits, error) {
	path := fmt.Sprintf("custom/%s/%s", hardtypes.QuerierRoute, hardtypes.QueryGetDeposits)

	page := 1
	var deposits hardtypes.Deposits

	for {
		params := hardtypes.NewQueryDepositsParams(page, c.PageLimit, "", sdk.AccAddress{})

		bz, err := c.cdc.MarshalJSON(&params)
		if err != nil {
			return nil, err
		}

		data, err := c.abciQuery(path, bz, height)
		if err != nil {
			return nil, err
		}

		var pagedDeposits hardtypes.Deposits
		err = c.cdc.UnmarshalJSON(data, &pagedDeposits)
		if err != nil {
			return nil, err
		}

		if len(pagedDeposits) > 0 {
			deposits = append(deposits, pagedDeposits...)
		}

		if len(pagedDeposits) < c.PageLimit {
			return deposits, nil
		}

		page++
	}
}

func (c *RpcLiquidationClient) abciQuery(
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

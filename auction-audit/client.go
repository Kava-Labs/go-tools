package main

import (
	"context"
	"errors"
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	auctiontypes "github.com/kava-labs/kava/x/auction/types"
	pricefeedtypes "github.com/kava-labs/kava/x/pricefeed/types"
	"github.com/tendermint/tendermint/libs/bytes"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	tmclient "github.com/tendermint/tendermint/rpc/client"
	rpchttpclient "github.com/tendermint/tendermint/rpc/client/http"
	coretypes "github.com/tendermint/tendermint/rpc/core/types"
)

type TmQuerier interface {
	tmclient.SignClient
	tmclient.ABCIClient
}

type Client struct {
	cdc *codec.Codec
	// rpc client for tendermint rpc
	Tendermint TmQuerier
}

func NewClient(
	rpcTarget string,
	cdc *codec.Codec,
) (Client, error) {
	rpcClient, err := rpchttpclient.New(rpcTarget, "/websocket")
	if err != nil {
		return Client{}, err
	}

	return Client{
		cdc:        cdc,
		Tendermint: rpcClient,
	}, nil
}

func (c Client) GetBeginBlockEventsFromQuery(
	ctx context.Context,
	query string,
) (sdk.StringEvents, int64, error) {
	// 1) Block search to find auction_start event and corresponding height
	// https://rpc.kava.io/block_search?query=%22auction_start.auction_id=16837%22

	blocks, err := c.QueryBlock(ctx, query)
	if err != nil {
		return nil, 0, err
	}

	if len(blocks) == 0 {
		return nil, 0, fmt.Errorf("no blocks found")
	}

	// 2) Block results to query events from height
	// https://rpc.kava.io/block_results?height=3146803
	events, err := c.GetBeginBlockEvents(ctx, blocks[0].Block.Height)
	return events, blocks[0].Block.Height, err
}

func (c Client) QueryBlock(ctx context.Context, query string) ([]*coretypes.ResultBlock, error) {
	page := 1
	perPage := 100

	res, err := c.Tendermint.BlockSearch(
		ctx,
		query,
		&page,
		&perPage,
		"desc",
	)

	if err != nil {
		return nil, fmt.Errorf("failed BlockSearch: %w", err)
	}

	return res.Blocks, nil
}

func (c Client) GetBeginBlockEvents(ctx context.Context, height int64) (sdk.StringEvents, error) {
	res, err := c.Tendermint.BlockResults(
		&height,
	)

	if err != nil {
		return nil, fmt.Errorf("failed BlockResults: %w", err)
	}

	// Do not use sdk.StringifyEvents as it flattens events which makes it
	// more difficult to parse.
	strEvents := make(sdk.StringEvents, 0, len(res.BeginBlockEvents))
	for _, e := range res.BeginBlockEvents {
		strEvents = append(strEvents, sdk.StringifyEvent(e))
	}

	return strEvents, nil
}

// GetPriceAtHeight returns the price of the given market at the given height.
func (c Client) GetPriceAtHeight(height int64, marketID string) (sdk.Dec, error) {
	bz, err := c.cdc.MarshalJSON(pricefeedtypes.NewQueryWithMarketIDParams(marketID))
	if err != nil {
		return sdk.ZeroDec(), err
	}

	path := fmt.Sprintf("custom/%s/%s", pricefeedtypes.QuerierRoute, pricefeedtypes.QueryPrice)

	data, err := c.abciQuery(path, bz, height)
	if err != nil {
		return sdk.ZeroDec(), err
	}

	var currentPrice pricefeedtypes.CurrentPrice
	err = c.cdc.UnmarshalJSON(data, &currentPrice)
	if err != nil {
		return sdk.ZeroDec(), err
	}

	return currentPrice.Price, nil
}

// GetAuction returns the price of the given market at the given height.
func (c Client) GetAuction(height int64, auctionID uint64) (auctiontypes.Auction, error) {
	bz, err := c.cdc.MarshalJSON(auctiontypes.NewQueryAuctionParams(auctionID))
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("custom/%s/%s", auctiontypes.QuerierRoute, auctiontypes.QueryGetAuction)

	data, err := c.abciQuery(path, bz, height)
	if err != nil {
		return nil, err
	}

	var auction auctiontypes.Auction
	err = c.cdc.UnmarshalJSON(data, &auction)
	if err != nil {
		return nil, err
	}

	return auction, nil
}

func (c Client) abciQuery(
	path string,
	data bytes.HexBytes,
	height int64) ([]byte, error) {
	opts := rpcclient.ABCIQueryOptions{Height: height, Prove: false}

	result, err := c.Tendermint.ABCIQueryWithOptions(path, data, opts)
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

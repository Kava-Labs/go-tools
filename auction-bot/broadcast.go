package main

import (
	"fmt"

	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

type BroadcastClient interface {
	GetChainID() (string, error)
	GetAccount(addr sdk.AccAddress) (*authtypes.BaseAccount, error)
	EstimateGas(tx *authtypes.StdTx) (uint64, error)
	BroadcastTxSync(tx *authtypes.StdTx) (*ctypes.ResultBroadcastTx, error)
}

type RpcBroadcastClient struct {
	rpc RpcClient
	cdc *codec.Codec
}

func NewRpcBroadcastClient(rpc RpcClient, cdc *codec.Codec) *RpcBroadcastClient {
	return &RpcBroadcastClient{
		rpc: rpc,
		cdc: cdc,
	}
}

func (c *RpcBroadcastClient) GetChainID() (string, error) {
	result, err := c.rpc.Status()
	if err != nil {
		return "", err
	}

	return result.NodeInfo.Network, nil
}

func (c *RpcBroadcastClient) GetAccount(address sdk.AccAddress) (*authtypes.BaseAccount, error) {
	params := authtypes.NewQueryAccountParams(address)

	bz, err := c.cdc.MarshalJSON(params)
	if err != nil {
		return nil, err
	}

	path := fmt.Sprintf("custom/acc/account/%s", address.String())

	result, err := ParseABCIResult(c.rpc.ABCIQuery(path, bz))
	if err != nil {
		return nil, err
	}

	var acc authtypes.BaseAccount
	err = c.cdc.UnmarshalJSON(result, &acc)
	if err != nil {
		return nil, err
	}

	return &acc, err
}

func (c *RpcBroadcastClient) EstimateGas(tx *authtypes.StdTx) (uint64, error) {
	txBz, err := c.cdc.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		return 0, err
	}

	bz, err := ParseABCIResult(c.rpc.ABCIQuery("/app/simulate", txBz))
	if err != nil {
		return 0, err
	}

	var simRes sdk.SimulationResponse
	if err := c.cdc.UnmarshalBinaryBare(bz, &simRes); err != nil {
		return 0, err
	}

	return simRes.GasInfo.GasUsed, nil
}

func (c *RpcBroadcastClient) BroadcastTxSync(tx *authtypes.StdTx) (*ctypes.ResultBroadcastTx, error) {
	txBz, err := c.cdc.MarshalBinaryLengthPrefixed(tx)
	if err != nil {
		return nil, err
	}

	return c.rpc.BroadcastTxSync(txBz)
}

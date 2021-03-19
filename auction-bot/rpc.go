package main

import (
	"errors"

	"github.com/tendermint/tendermint/libs/bytes"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type RpcClient interface {
	Status() (*ctypes.ResultStatus, error)
	ABCIQuery(
		path string,
		data bytes.HexBytes,
	) (*ctypes.ResultABCIQuery, error)
	ABCIQueryWithOptions(
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) (*ctypes.ResultABCIQuery, error)
	BroadcastTxSync(tx tmtypes.Tx) (*ctypes.ResultBroadcastTx, error)
}

func ParseABCIResult(result *ctypes.ResultABCIQuery, err error) ([]byte, error) {
	if err != nil {
		return []byte{}, err
	}

	resp := result.Response
	if !resp.IsOK() {
		return []byte{}, errors.New(resp.Log)
	}

	value := result.Response.GetValue()
	if value == nil {
		return []byte{}, nil
	}

	return value, nil
}
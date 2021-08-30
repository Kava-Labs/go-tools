package rpc

import (
	"github.com/tendermint/tendermint/libs/bytes"
	rpcclient "github.com/tendermint/tendermint/rpc/client"
	ctypes "github.com/tendermint/tendermint/rpc/core/types"
)

// RpcClient defines an interface that can query information via ABCI
type RpcClient interface {
	Status() (*ctypes.ResultStatus, error)
	ABCIQueryWithOptions(
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) (*ctypes.ResultABCIQuery, error)
}

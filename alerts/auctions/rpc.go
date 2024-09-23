package auctions

import (
	"context"
	"errors"

	"github.com/cometbft/cometbft/libs/bytes"
	rpcclient "github.com/cometbft/cometbft/rpc/client"
	ctypes "github.com/cometbft/cometbft/rpc/core/types"
)

type RpcClient interface {
	Status(ctx context.Context) (*ctypes.ResultStatus, error)
	ABCIQueryWithOptions(
		ctx context.Context,
		path string,
		data bytes.HexBytes,
		opts rpcclient.ABCIQueryOptions,
	) (*ctypes.ResultABCIQuery, error)
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

package claim

import (
	"fmt"

	bnbRpc "github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/types"
)

// querySwapsMaxPageSize is the maximum supported 'limit' parameter for querying swaps by recipient address
const querySwapsMaxPageSize = 100

type bnbChainClient interface {
	// TODO
}

var _ bnbChainClient = rcpBNBClient{}

type rcpBNBClient struct {
	deputyAddressString string
	bnbSDKClient        *bnbRpc.HTTP
}

func newRpcBNBClient(rpcURL string, deputyAddress string) rcpBNBClient {
	return rcpBNBClient{
		deputyAddressString: deputyAddress,
		bnbSDKClient:        bnbRpc.NewRPCClient(rpcURL, types.ProdNetwork),
	}
}

func (bc rcpBNBClient) getOpenSwaps() ([]types.AtomicSwap, error) {
	var swapIDs []types.SwapBytes
	var queryOffset int64 = 0
	for {
		swapIDsPage, err := bc.bnbSDKClient.GetSwapByRecipient(bc.deputyAddressString, queryOffset, querySwapsMaxPageSize)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch swaps by recipient (offset %d): %w", queryOffset, err)
		}
		swapIDs = append(swapIDs, swapIDsPage...)
		if len(swapIDsPage) < querySwapsMaxPageSize {
			// if less than a full page of swapIDs was returned, there is no more to query
			break
		}
		queryOffset += querySwapsMaxPageSize
	}

	var swaps []types.AtomicSwap
	for _, id := range swapIDs {
		s, err := bc.getSwapByID(id)
		if err != nil {
			return nil, fmt.Errorf("couldn't find swap for ID %x: %w", id, err) // TODO should probably retry on failure
		}
		if s.Status != types.Open {
			continue
		}
		swaps = append(swaps, s)
	}
	return swaps, nil
}

func (bc rcpBNBClient) getAccount(address types.AccAddress) (types.Account, error) {
	return bc.bnbSDKClient.GetAccount(address)
}

func (bc rcpBNBClient) getTxConfirmation(txHash []byte) (*bnbRpc.ResultTx, error) {
	return bc.bnbSDKClient.Tx(txHash, false)
}

func (bc rcpBNBClient) getSwapByID(id types.SwapBytes) (types.AtomicSwap, error) {
	return bc.bnbSDKClient.GetSwapByID(id)
}

package claim

import (
	bnbRpc "github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/types"
)

// querySwapsMaxPageSize is the maximum supported 'limit' parameter for querying swaps by recipient address
const querySwapsMaxPageSize = 100

type bnbChainClient struct {
	deputyAddressString string
	bnbSDKClient        *bnbRpc.HTTP
}

func newBnbChainClient(rpcURL string, deputyAddress string) bnbChainClient {
	return bnbChainClient{
		deputyAddressString: deputyAddress,
		bnbSDKClient:        bnbRpc.NewRPCClient(rpcURL, types.ProdNetwork),
	}
}

func (bc bnbChainClient) getOpenSwaps() ([]types.AtomicSwap, error) {
	var swapIDs []types.SwapBytes
	var queryOffset int64 = 0
	for {
		swapIDsPage, err := bc.bnbSDKClient.GetSwapByRecipient(bc.deputyAddressString, queryOffset, querySwapsMaxPageSize)
		if err != nil {
			return nil, err
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
			return nil, err // TODO should probably retry on failure
		}
		if s.Status != types.Open {
			continue
		}
		swaps = append(swaps, s)
	}
	return swaps, nil
}

func (bc bnbChainClient) getAccount(address types.AccAddress) (types.Account, error) {
	return bc.bnbSDKClient.GetAccount(address)
}

func (bc bnbChainClient) getTxConfirmation(txHash []byte) (*bnbRpc.ResultTx, error) {
	return bc.bnbSDKClient.Tx(txHash, false)
}

func (bc bnbChainClient) getSwapByID(id types.SwapBytes) (types.AtomicSwap, error) {
	return bc.bnbSDKClient.GetSwapByID(id)
}

package claim

import (
	"fmt"

	bnbRpc "github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/types"
)

// querySwapsMaxPageSize is the maximum supported 'limit' parameter for querying swaps by recipient address
const querySwapsMaxPageSize = 100

//go:generate mockgen -destination mock/bnb_client.go -package mock . BnbChainClient

type BnbChainClient interface { // XXX should be defined in the claimer, not the client. Doesn't need to be exported?
	GetTxConfirmation(txHash []byte) (*bnbRpc.ResultTx, error)
	GetOpenSwaps() ([]types.AtomicSwap, error)
	GetBNBSDKClient() *bnbRpc.HTTP
	GetSwapByID(id types.SwapBytes) (types.AtomicSwap, error)
}

var _ BnbChainClient = rpcBNBClient{}

type rpcBNBClient struct {
	deputyAddressString string
	bnbSDKClient        *bnbRpc.HTTP
}

func NewRpcBNBClient(rpcURL string, deputyAddress string) rpcBNBClient {
	return rpcBNBClient{
		deputyAddressString: deputyAddress,
		bnbSDKClient:        bnbRpc.NewRPCClient(rpcURL, types.ProdNetwork),
	}
}

func (bc rpcBNBClient) GetOpenSwaps() ([]types.AtomicSwap, error) {
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
		s, err := bc.GetSwapByID(id)
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

func (bc rpcBNBClient) GetAccount(address types.AccAddress) (types.Account, error) {
	return bc.bnbSDKClient.GetAccount(address)
}

func (bc rpcBNBClient) GetTxConfirmation(txHash []byte) (*bnbRpc.ResultTx, error) {
	return bc.bnbSDKClient.Tx(txHash, false)
}

func (bc rpcBNBClient) GetSwapByID(id types.SwapBytes) (types.AtomicSwap, error) {
	return bc.bnbSDKClient.GetSwapByID(id)
}

func (bc rpcBNBClient) GetBNBSDKClient() *bnbRpc.HTTP {
	return bc.bnbSDKClient
}

package claim

import (
	bnbRpc "github.com/binance-chain/go-sdk/client/rpc"
	"github.com/binance-chain/go-sdk/common/types"
)

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
	swapIDs, err := bc.bnbSDKClient.GetSwapByRecipient(bc.deputyAddressString, 0, 100) // TODO handle limits
	if err != nil {
		return nil, err
	}

	var swaps []types.AtomicSwap
	for _, id := range swapIDs {
		s, err := bc.getSwapByID(id)
		if err != nil {
			return nil, err // TODO should probably retry on failure
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

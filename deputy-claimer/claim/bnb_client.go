package claim

import (
	"fmt"

	bnbRpc "github.com/kava-labs/binance-chain-go-sdk/client/rpc"
	"github.com/kava-labs/binance-chain-go-sdk/common/types"
	bnbtypes "github.com/kava-labs/binance-chain-go-sdk/types"
	ctypes "github.com/kava-labs/tendermint/rpc/core/types"
	tmtypes "github.com/kava-labs/tendermint/types"
)

// querySwapsMaxPageSize is the maximum supported 'limit' parameter for querying swaps by recipient address
const querySwapsMaxPageSize = 100

//go:generate mockgen -destination mock/bnb_client.go -package mock . BnbChainClient

type BnbChainClient interface { // XXX should be defined in the claimer, not the client. Doesn't need to be exported?
	GetOpenOutgoingSwaps() ([]types.AtomicSwap, error)
	GetRandomNumberFromSwap(id []byte) ([]byte, error)
	GetTxConfirmation(txHash []byte) (*bnbRpc.ResultTx, error)
	GetAccount(address types.AccAddress) (types.Account, error)
	GetChainID() string
	BroadcastTx(tx tmtypes.Tx) error
	Status() (*ctypes.ResultStatus, error)
}

var _ BnbChainClient = rpcBNBClient{}

type rpcBNBClient struct {
	deputyAddresses []types.AccAddress
	bnbSDKClient    *bnbRpc.HTTP
}

func NewRpcBNBClient(rpcURL string, deputyAddresses []types.AccAddress) rpcBNBClient {
	return rpcBNBClient{
		deputyAddresses: deputyAddresses,
		bnbSDKClient:    bnbRpc.NewRPCClient(rpcURL, types.ProdNetwork),
	}
}

func (bc rpcBNBClient) GetOpenOutgoingSwaps() ([]types.AtomicSwap, error) {
	var swapIDs []types.SwapBytes
	for _, addr := range bc.deputyAddresses {
		ids, err := bc.GetSwapIDsByRecipient(addr)
		if err != nil {
			return nil, fmt.Errorf("failed to get swaps for deputy %s", addr)
		}
		swapIDs = append(swapIDs, ids...)
	}

	var swaps []types.AtomicSwap
	for _, id := range swapIDs {
		s, err := bc.bnbSDKClient.GetSwapByID(id)
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

func (bc rpcBNBClient) GetSwapIDsByRecipient(recipient types.AccAddress) ([]types.SwapBytes, error) {
	var swapIDs []types.SwapBytes
	var queryOffset int64 = 0
	for {
		swapIDsPage, err := bc.bnbSDKClient.GetSwapByRecipient(recipient.String(), queryOffset, querySwapsMaxPageSize)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch swaps by recipient (offset %d): %w", queryOffset, err) // TODO should probably retry on failure
		}
		swapIDs = append(swapIDs, swapIDsPage...)
		if len(swapIDsPage) < querySwapsMaxPageSize {
			// if less than a full page of swapIDs was returned, there is no more to query
			break
		}
		queryOffset += querySwapsMaxPageSize
	}
	return swapIDs, nil
}

func (bc rpcBNBClient) GetRandomNumberFromSwap(id []byte) ([]byte, error) {
	swap, err := bc.bnbSDKClient.GetSwapByID(id)
	if err != nil {
		return nil, fmt.Errorf("could not fetch swap: %w", err)
	}
	if len(swap.RandomNumber) == 0 {
		return nil, fmt.Errorf("found swap without random number, status %s", swap.Status)
	}
	return swap.RandomNumber, nil
}

func (bc rpcBNBClient) GetTxConfirmation(txHash []byte) (*bnbRpc.ResultTx, error) {
	return bc.bnbSDKClient.Tx(txHash, false)
}

func (bc rpcBNBClient) GetAccount(address types.AccAddress) (types.Account, error) {
	return bc.bnbSDKClient.GetAccount(address)
}

func (bc rpcBNBClient) GetChainID() string {
	// this could fetch the chain id from the node, but it's unlikely to ever change
	return bnbtypes.ProdChainID
}

func (bc rpcBNBClient) BroadcastTx(tx tmtypes.Tx) error {
	res, err := bc.bnbSDKClient.BroadcastTxSync(tx)
	if err != nil {
		return err
	}
	if res.Code != 0 { // tx failed to be submitted to the mempool
		return fmt.Errorf("transaction failed to get into mempool: %s", res.Log)
	}
	return nil
}

func (bc rpcBNBClient) Status() (*ctypes.ResultStatus, error) {
	return bc.bnbSDKClient.Status()
}
